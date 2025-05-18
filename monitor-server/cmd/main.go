package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"monitor-server/internal/config"
	"monitor-server/internal/handle/gin"
	"monitor-server/internal/repository"
	"monitor-server/internal/repository/redis"
	"monitor-server/internal/service"
	pb "monitor-server/proto"

	"google.golang.org/grpc"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	tdengineRepo, err := repository.NewTDengineRepository(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to TDengine: %v", err)
	}

	// 初始化 Redis 仓库
	redisRepo, err := repository.NewRedisRepository(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Redis repository: %v", err)
	}

	// 创建服务实例
	monitorService := service.NewMonitorService(tdengineRepo, redisRepo)

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer()
	pb.RegisterMonitorServiceServer(grpcServer, monitorService)

	// 启动 gRPC 服务器
	go func() {
		addr := "0.0.0.0:" + cfg.Grpc.Port
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}
		log.Println("gRPC server starting on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// 设置 HTTP 路由
	r := gin.SetupRouter(redisRepo)

	// 启动 HTTP 服务器
	go func() {
		addr := "0.0.0.0:" + cfg.Server.Port // 使用固定端口或从配置中读取
		log.Printf("HTTP server starting on %s", addr)
		if err := r.Run(addr); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 创建上下文，用于优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 启动定时清理任务
	go redis.StartCleanupTask(ctx, redisRepo, tdengineRepo)

	// 等待中断信号
	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	// 优雅关闭 gRPC 服务器
	grpcServer.GracefulStop()
	log.Println("Server stopped")

}
