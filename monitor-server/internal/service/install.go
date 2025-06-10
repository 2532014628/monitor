package service

import (
	"context"
	"fmt"
	"monitor-server/internal/repository"
	pb "monitor-server/proto"

	"golang.org/x/crypto/ssh"
)

type InstallService struct {
	tdengineRepo *repository.TDengineRepository
	redisRepo    *repository.RedisRepository
}

func NewInstallService(tdengineRepo *repository.TDengineRepository, redisRepo *repository.RedisRepository) *InstallService {
	return &InstallService{
		tdengineRepo: tdengineRepo,
		redisRepo:    redisRepo,
	}
}

// InstallAgent 安装agent
func (s *InstallService) InstallAgent(ctx context.Context, req *pb.InstallAgentRequest) (*pb.InstallAgentResponse, error) {
	// 将阈值存入Redis
	memKey := fmt.Sprintf("mem_threshold:%s", req.HostName)
	cpuKey := fmt.Sprintf("cpu_threshold:%s", req.HostName)

	if err := s.redisRepo.Set(ctx, memKey, req.MemThreshold, 0); err != nil {
		return nil, fmt.Errorf("存储内存阈值失败: %v", err)
	}

	if err := s.redisRepo.Set(ctx, cpuKey, req.CpuThreshold, 0); err != nil {
		return nil, fmt.Errorf("存储CPU阈值失败: %v", err)
	}

	// 安装agent
	err := s.doInstallAgent(req, req.Token)
	if err != nil {
		return nil, fmt.Errorf("安装agent失败: %v", err)
	}

	return &pb.InstallAgentResponse{
		Success: true,
		Message: "Agent安装成功",
	}, nil
}

// 执行agent安装
func (s *InstallService) doInstallAgent(req *pb.InstallAgentRequest, token string) error {
	// SSH配置
	config := &ssh.ClientConfig{
		User: req.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 建立SSH连接
	addr := fmt.Sprintf("%s:%d", req.Host, req.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer client.Close()

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建会话失败: %v", err)
	}
	defer session.Close()

	// 准备安装命令
	packageCmd := ""
	switch req.Platform {
	case "ubuntu", "debian":
		packageCmd = "apt update && apt install -y git"
	case "centos", "rhel", "fedora":
		packageCmd = "yum install -y git"
	default:
		return fmt.Errorf("不支持的操作系统: %s", req.Platform)
	}

	// 构建安装脚本
	cmd := fmt.Sprintf(`
#!/bin/bash
# 安装git
%s

# 克隆代码仓库
git clone https://gitee.com/wu-jinhao111/agent.git
cd agent/agent || exit

# 创建systemd服务文件
cat <<EOF | sudo tee /etc/systemd/system/agent.service
[Unit]
Description=Agent Service
After=network.target

[Service]
Type=simple
User=%s
WorkingDirectory=%s/agent/agent
ExecStart=%s/agent/agent/main -hostname=%s -token=%s
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 重新加载systemd配置
sudo systemctl daemon-reload

# 启用并启动服务
sudo systemctl enable agent.service
sudo systemctl start agent.service

# 检查服务状态
sudo systemctl status agent.service
`, packageCmd, req.User, req.User, req.User, req.HostName, token)

	// 执行命令
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("执行安装命令失败: %v", err)
	}

	return nil
}

// UninstallAgent 删除agent
func (s *InstallService) UninstallAgent(ctx context.Context, req *pb.UninstallAgentRequest) (*pb.UninstallAgentResponse, error) {
	// 删除agent
	err := s.doUninstallAgent(req)
	if err != nil {
		return nil, fmt.Errorf("删除agent失败: %v", err)
	}

	// 清理数据库中的相关数据
	if err := s.tdengineRepo.CleanHostData(ctx, req.Hostname); err != nil {
		return nil, fmt.Errorf("清理数据库数据失败: %v", err)
	}

	// 清理Redis中的相关数据
	if err := s.redisRepo.CleanHostData(ctx, req.Hostname); err != nil {
		return nil, fmt.Errorf("清理Redis数据失败: %v", err)
	}

	return &pb.UninstallAgentResponse{
		Success: true,
		Message: "Agent删除成功",
	}, nil
}

// 执行agent删除
func (s *InstallService) doUninstallAgent(req *pb.UninstallAgentRequest) error {
	// SSH配置
	config := &ssh.ClientConfig{
		User: req.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 建立SSH连接
	addr := fmt.Sprintf("%s:%d", req.Host, req.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}
	defer client.Close()

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建会话失败: %v", err)
	}
	defer session.Close()

	// 构建删除命令
	cmd := `
#!/bin/bash
# 停止并禁用服务
sudo systemctl stop agent.service
sudo systemctl disable agent.service

# 删除服务文件
sudo rm -f /etc/systemd/system/agent.service

# 重新加载systemd配置
sudo systemctl daemon-reload

# 删除agent目录
sudo rm -rf ~/agent

# 检查服务是否已删除
if ! systemctl is-active --quiet agent.service; then
    echo "Agent服务已成功删除"
else
    echo "Agent服务删除失败"
    exit 1
fi
`

	// 执行命令
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("执行删除命令失败: %v", err)
	}

	return nil
}
