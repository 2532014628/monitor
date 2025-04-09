package install

import (
	"backend/server/model"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
)

type SshInfo struct {
	Host       string `json:"host"`
	User       string `json:"user"`
	Password   string `json:"password"`
	Port       int    `json:"port"`
	Host_Name  string `json:"host_name"`
	OS         string `json:"os"`
	Platform   string `json:"platform"`
	KernelArch string `json:"kernel_arch"`
	Token      string `json:"token"`
}

// InstallAgent 安装agent
func InstallAgent(c *gin.Context) {
	Username, exists := c.Get("username")
	if !exists {
		log.Printf("未找到用户名")
		c.JSON(401, gin.H{
			"code":    401,
			"success": false,
			"message": "未找到用户信息",
		})
		return
	}
	username := Username.(string)
	// 解析json body 到结构体 SshInfo
	var agentInfo SshInfo
	if err := c.BindJSON(&agentInfo); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查数据库中是否存在相同的 host_name
	var exist bool
	query := `SELECT EXISTS (SELECT 1 FROM host_info WHERE host_name = $1)`
	err := model.DB.QueryRow(query, agentInfo.Host_Name).Scan(&exist)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to check host_name in database"})
		return
	}

	// 如果 host_name 已存在，返回错误并停止安装
	if exist {
		c.IndentedJSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("host_name '%s' already exists", agentInfo.Host_Name)})
		return
	}

	// 生成16位随机token
	token, err := generateToken(16)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	agentInfo.Token = token
	// 插入 host_info 表
	var hostInfo model.HostInfo
	hostInfo.Hostname = agentInfo.Host_Name
	hostInfo.OS = agentInfo.OS
	hostInfo.Platform = agentInfo.Platform
	hostInfo.KernelArch = agentInfo.KernelArch
	hostInfo.Token = agentInfo.Token
	hostInfo.CreatedAt = time.Now()
	err = model.InsertHostInfo(hostInfo, username)
	if err != nil {
		s := fmt.Sprintf("Failed to insert host info: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": s})
		return
	}

	// 存储host_name和token到数据库
	err = model.InsertHostandToken(agentInfo.Host_Name, agentInfo.Token)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert host info into database"})
		return
	}

	// 安装agent
	err = DoInstallAgent(agentInfo)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 安装成功，返回成功信息
	c.IndentedJSON(http.StatusOK, gin.H{"message": "Agent installed successfully", "host_name": agentInfo.Host_Name, "token": agentInfo.Token})
}

// 随机生成指定长度的随机token
func generateToken(length int) (string, error) {
	bytes := make([]byte, length/2) // 16字节 = 16位字符
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// DoInstallAgent 执行 agent 安装
func DoInstallAgent(ss SshInfo) error {
	// SSH 配置
	config := &ssh.ClientConfig{
		User: ss.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(ss.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 建立 SSH 连接
	s := fmt.Sprintf("%s:%v", ss.Host, ss.Port)
	client, err := ssh.Dial("tcp", s, config)
	if err != nil {
		fmt.Printf("Failed to dial: %s", err)
		return err
	}
	defer client.Close()

	// 创建新会话
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session: %s", err)
		return err
	}
	defer session.Close()
	packageCmd := ""
	switch ss.Platform {
	case "ubuntu", "debian":
		packageCmd = "apt update && apt install -y git"
	case "centos", "rhel", "fedora":
		packageCmd = "yum install -y git"
	default:
		return fmt.Errorf("unsupported platform: %s", ss.Platform)
	}
	cmd := fmt.Sprintf(
		`#!/bin/bash
		# 克隆代码仓库
        %s

		git clone https://gitee.com/wu-jinhao111/agent.git
		cd agent/agent || exit

		# 授予执行权限并运行主程序
		chmod +x main
		./main -host_name="%s" -token="%s" &

		# 创建 systemd 服务文件
		cat <<EOF | sudo tee /etc/systemd/system/main_startup.service
		[Unit]
		Description=Main Program Startup Service
		After=network.target

		[Service]
		Type=simple
		ExecStart=$HOME/agent/agent/main -host_name=%s -token=%s
		Restart=always

		[Install]
		WantedBy=multi-user.target
		EOF

		# 启用并启动服务
		sudo systemctl enable main_startup.service
		sudo systemctl start main_startup.service
		`,
		packageCmd, ss.Host_Name, ss.Token, ss.Host_Name, ss.Token)

	// 运行命令
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		fmt.Printf("Failed to run command: %s", err)
		return err
	}
	fmt.Println(string(output))

	return nil
}
