// internal/handler/grpc/monitor.go
package grpc

import (
	"context"
	"monitor-server/internal/service"
	pb "monitor-server/proto"
)

type MonitorHandler struct {
	pb.UnimplementedMonitorServiceServer
	monitorService *service.MonitorService
	installService *service.InstallService
}

func NewMonitorHandler(monitorService *service.MonitorService, installService *service.InstallService) *MonitorHandler {
	return &MonitorHandler{
		monitorService: monitorService,
		installService: installService,
	}
}

// GetSystemInfo 获取系统信息
func (h *MonitorHandler) GetSystemInfo(ctx context.Context, req *pb.GetSystemInfoRequest) (*pb.GetSystemInfoResponse, error) {
	return h.monitorService.GetSystemInfo(ctx, req)
}

// GetLastSystemInfo 获取最新的系统信息
func (h *MonitorHandler) GetLastSystemInfo(ctx context.Context, req *pb.GetLastSystemInfoRequest) (*pb.GetLastSystemInfoResponse, error) {
	return h.monitorService.GetLastSystemInfo(ctx, req)
}
