package grpc

import (
	"context"
	pb "monitor-server/proto"
)

// InstallAgent 安装agent
func (h *MonitorHandler) InstallAgent(ctx context.Context, req *pb.InstallAgentRequest) (*pb.InstallAgentResponse, error) {
	// 调用安装服务进行安装
	return h.installService.InstallAgent(ctx, req)
}

// ListAgents 获取agent列表
func (h *MonitorHandler) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	// 调用监控服务获取agent列表
	return h.monitorService.ListAgents(ctx, req)
}

// CheckServerStatus 检查服务器状态
func (h *MonitorHandler) CheckServerStatus(ctx context.Context, req *pb.CheckServerStatusRequest) (*pb.CheckServerStatusResponse, error) {
	// 调用监控服务检查服务器状态
	return h.monitorService.CheckServerStatus(ctx, req)
}
