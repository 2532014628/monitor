// internal/service/monitor.go
package service

import (
	"context"
	"database/sql"

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
	info, err := s.tdengineRepo.GetLastSystemInfo(ctx, req.Hostname)
	if err != nil {
		return nil, err
	}
	return &pb.GetLastSystemInfoResponse{
		Info: info,
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
