// internal/service/monitor.go
package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"monitor-server/internal/model"
	"monitor-server/internal/repository"
	pb "monitor-server/proto"
)

type MonitorService struct {
	pb.UnimplementedMonitorServiceServer
	db           *sql.DB
	tdengineRepo *repository.TDengineRepository
	redisRepo    *repository.RedisRepository
}

func NewMonitorService(tdengineRepo *repository.TDengineRepository, redisRepo *repository.RedisRepository) *MonitorService {
	return &MonitorService{
		tdengineRepo: tdengineRepo,
		redisRepo:    redisRepo,
	}
}

// GetSystemInfo 获取系统信息
func (s *MonitorService) GetSystemInfo(ctx context.Context, req *pb.GetSystemInfoRequest) (*pb.GetSystemInfoResponse, error) {
	info, err := s.tdengineRepo.GetSystemInfo(ctx, req.Hostname, req.From, req.To)
	if err != nil {
		return nil, err
	}
	return &pb.GetSystemInfoResponse{
		Info: info,
	}, nil
}

// GetLastSystemInfo 获取最新的系统信息
func (s *MonitorService) GetLastSystemInfo(ctx context.Context, req *pb.GetLastSystemInfoRequest) (*pb.GetLastSystemInfoResponse, error) {
	// 先从Redis获取最新系统信息
	pattern := fmt.Sprintf("system_info:%s:*", req.Hostname)
	keys, _, err := s.redisRepo.Scan(ctx, 0, pattern, 1)
	if err != nil {
		return nil, fmt.Errorf("扫描Redis键失败: %v", err)
	}

	var info *pb.SystemInfo
	if len(keys) > 0 {
		// 获取最新的键
		latestKey := keys[len(keys)-1]
		baseKey := latestKey[:strings.LastIndex(latestKey, ":")]

		// 获取主机信息
		var hostInfo model.HostInfo
		if err := s.redisRepo.Get(ctx, baseKey+":host", &hostInfo); err != nil {
			return nil, fmt.Errorf("从Redis获取主机信息失败: %v", err)
		}

		// 获取CPU信息
		var cpuInfo []model.CPUInfo
		if err := s.redisRepo.Get(ctx, baseKey+":cpu", &cpuInfo); err != nil {
			return nil, fmt.Errorf("从Redis获取CPU信息失败: %v", err)
		}

		// 获取内存信息
		var memInfo model.MemoryInfo
		if err := s.redisRepo.Get(ctx, baseKey+":mem", &memInfo); err != nil {
			return nil, fmt.Errorf("从Redis获取内存信息失败: %v", err)
		}

		// 获取网络信息
		var netInfo []model.NetworkInfo
		if err := s.redisRepo.Get(ctx, baseKey+":net", &netInfo); err != nil {
			return nil, fmt.Errorf("从Redis获取网络信息失败: %v", err)
		}

		// 转换为 SystemInfo
		info = &pb.SystemInfo{
			HostInfo: &pb.HostInfo{
				Id:         int32(hostInfo.ID),
				HostName:   hostInfo.Hostname,
				Os:         hostInfo.OS,
				Platform:   hostInfo.Platform,
				KernelArch: hostInfo.KernelArch,
				CreatedAt:  hostInfo.CreatedAt.Format(time.RFC3339),
			},
			MemoryInfo: &pb.MemoryInfo{
				Id:          int32(memInfo.ID),
				Total:       memInfo.Total,
				Available:   memInfo.Available,
				Used:        memInfo.Used,
				Free:        memInfo.Free,
				UserPercent: memInfo.UserPercent,
				CreatedAt:   memInfo.CreatedAt.Format(time.RFC3339),
			},
		}

		// 转换CPU信息
		for _, cpu := range cpuInfo {
			info.CpuInfo = append(info.CpuInfo, &pb.CPUInfo{
				Id:        int32(cpu.ID),
				ModelName: cpu.ModelName,
				CoresNum:  int32(cpu.CoresNum),
				Percent:   cpu.Percent,
				CreatedAt: cpu.CreatedAt.Format(time.RFC3339),
			})
		}

		// 转换网络信息
		for _, net := range netInfo {
			info.NetworkInfo = append(info.NetworkInfo, &pb.NetworkInfo{
				Id:        int32(net.ID),
				Name:      net.Name,
				BytesRecv: net.BytesRecv,
				BytesSent: net.BytesSent,
				CreatedAt: net.CreatedAt.Format(time.RFC3339),
			})
		}
	} else {
		// Redis中没有数据，从TDengine获取
		var err error
		info, err = s.tdengineRepo.GetLastSystemInfo(ctx, req.Hostname)
		if err != nil {
			return nil, err
		}
	}

	// 从Redis获取阈值
	memKey := fmt.Sprintf("mem_threshold:%s", req.Hostname)
	cpuKey := fmt.Sprintf("cpu_threshold:%s", req.Hostname)

	var memThreshold, cpuThreshold float64
	if err := s.redisRepo.Get(ctx, memKey, &memThreshold); err != nil {
		return nil, fmt.Errorf("获取内存阈值失败: %v", err)
	}
	if err := s.redisRepo.Get(ctx, cpuKey, &cpuThreshold); err != nil {
		return nil, fmt.Errorf("获取CPU阈值失败: %v", err)
	}

	// 检查告警
	var alertMessages []string

	// 检查CPU告警
	for _, cpu := range info.CpuInfo {
		if cpu.Percent > cpuThreshold {
			alertMessages = append(alertMessages, "CPU告警")
			break
		}
	}

	// 检查内存告警
	if info.MemoryInfo.UserPercent > memThreshold {
		alertMessages = append(alertMessages, "内存告警")
	}

	// 生成告警信息
	var alertMessage string
	if len(alertMessages) > 0 {
		alertMessage = strings.Join(alertMessages, "、")
	}

	return &pb.GetLastSystemInfoResponse{
		Info:         info,
		AlertMessage: alertMessage,
	}, nil
}

// GetHostLastUpdateTime 获取主机最后更新时间
func (s *MonitorService) GetHostLastUpdateTime(ctx context.Context, req *pb.GetHostLastUpdateTimeRequest) (*pb.GetHostLastUpdateTimeResponse, error) {
	// 构建 Redis key
	key := "host:" + req.Hostname

	// 从 Redis 获取最后更新时间
	lastUpdatedStr, err := s.redisRepo.GetLastUpdateTime(ctx, key)
	if err != nil {
		if err == repository.ErrKeyNotFound {
			return &pb.GetHostLastUpdateTimeResponse{
				Exists: false,
			}, nil
		}
		return nil, err
	}

	return &pb.GetHostLastUpdateTimeResponse{
		LastUpdated: lastUpdatedStr,
		Exists:      true,
	}, nil
}
