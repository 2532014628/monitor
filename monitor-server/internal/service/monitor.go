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
