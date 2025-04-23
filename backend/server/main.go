package main

import (
	"backend/server/config"
	"backend/server/handle/admin"
	"backend/server/handle/agent/install"
	"backend/server/handle/company"
	"backend/server/handle/server/monitor" // 引入 monitor 包
	"backend/server/handle/user/info"
	"backend/server/handle/user/login"
	"backend/server/handle/user/update"
	"backend/server/middlewire"
	"backend/server/middlewire/cors"
	db "backend/server/model/init"
	"backend/server/redis"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	//ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	//defer stop()

	//go func() {
	//	<-ctx.Done()
	//	// 执行连接关闭
	//	if err := model.DB.Close(); err != nil {
	//		log.Printf("Failed to close database connection: %v", err)
	//	}
	//	if err := model.TDengine.Close(); err != nil {
	//		log.Printf("Failed to close TDengine connection: %v", err)
	//	}
	//	os.Exit(0)
	//}()
	//读取DBConfig.yaml文件
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	//设置数据库连接的环境变量
	os.Setenv("DB_USER", config.DB.User)
	os.Setenv("DB_PASSWORD", config.DB.Password)
	os.Setenv("DB_HOST", config.DB.Host)
	os.Setenv("DB_PORT", config.DB.Port)
	os.Setenv("DB_NAME", config.DB.Name)
	//设置TDengine数据库连接的环境变量
	os.Setenv("TDENGINE_USER", config.TDengine.User)
	os.Setenv("TDENGINE_PASSWORD", config.TDengine.Password)
	os.Setenv("TDENGINE_HOST", config.TDengine.Host)
	os.Setenv("TDENGINE_PORT", config.TDengine.Port)
	os.Setenv("TDENGINE_NAME", config.TDengine.Name)
	// OSS服务
	os.Setenv("OSS_REGION", config.OSS.OSS_REGION)
	os.Setenv("OSS_ACCESS_KEY_ID", config.OSS.OSS_ACCESS_KEY_ID)
	os.Setenv("OSS_ACCESS_KEY_SECRET", config.OSS.OSS_ACCESS_KEY_SECRET)
	os.Setenv("OSS_BUCKET", config.OSS.OSS_BUCKET)
	// 邮箱服务
	os.Setenv("EMAIL_NAME", config.Email.Name)
	os.Setenv("EMAIL_PASSWORD", config.Email.Password)
	os.Setenv("BASE_URL", config.Email.Url)
	os.Setenv("SMTP_SERVER_HOST", config.SMTPServer.Host)
	os.Setenv("SMTP_SERVER_PORT", config.SMTPServer.Port)
	// Redis服务
	os.Setenv("REDIS_HOST", config.Redis.Host)
	os.Setenv("REDIS_PASSWORD", config.Redis.Password)
	os.Setenv("REDIS_PORT", config.Redis.Port)
	os.Setenv("REDIS_DB", config.Redis.DB)

	router := gin.Default()
	router.Use(cors.CORSMiddleware())
	// 初始化 Redis 连接
	if err := redis.InitRedis(); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	// 连接数据库
	if err := db.ConnectDatabase(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	//连接TDengine数据库
	if err := db.ConnectTDengine(); err != nil {
		log.Fatalf("Failed to connect to TDengine database: %v", err)
	}

	// 初始化数据库
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化数据库数据
	if err := db.InitDBData(); err != nil {
		log.Fatalf("Failed to initialize data: %v", err)
	}
	//初始化TDengine
	if err := db.InitTDengine(); err != nil {
		log.Fatalf("Failed to initialize TDengine: %v", err)
	}
	// 初始化redis
	// if err := db.InitRedis(); err!= nil {
	// 	log.Fatalf("Failed to connect to redis: %v", err)
	// }
	go monitor.CheckServerStatus()
	router.Static("/static", "./static")

	// 注册 Swagger 路由
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger.json")))

	router.POST("/agent/register", login.Register)
	router.POST("/agent/login", login.Login)
	// 需要 JWT 认证的路由
	auth := router.Group("/agent", middlewire.JWTAuthMiddleware())
	{
		// 用户信息
		auth.GET("/userInfo", info.GetUserInfo)
		auth.GET("/allUserInfo", info.GetAllUserInfo)
		auth.POST("/updateUserInfo", update.UpdateUserInfo)
		router.POST("/reset_password", update.ResetPassword)
		auth.POST("/request_reset_password", update.RequestResetPassword)
		auth.GET("/info/recivelist",info.GetReceiveList)

		// 系统/公司管理员操作
		auth.POST("/registercompany", company.Register)       //注册公司
		auth.POST("/addMember", admin.AddMember)              // 添加成员
		auth.POST("/deleteMembers", admin.DeleteMember)       // 批量删除成员
		auth.GET("/getmemberinfo", admin.GetMemberInfo)       // 获取公司成员信息
		auth.GET("/get-company-info", admin.GetCompanyInfo)   // 指定公司的信息（含成员信息）
		auth.GET("/get-company-list", company.GetCompanyList) // 公司列表
		auth.POST("/sshkey", admin.AddSShkey)                 // 添加SSH密钥
		auth.POST("/joincompany", admin.JoinCompany) 		  // 邀请成员加入公司
		auth.POST("/replaceadmin",admin.ReplaceAdmin)		  // 更换管理员

		// 监控
		auth.POST("/install", install.InstallAgent)
		auth.GET("/list", monitor.ListAgent)
		auth.GET("/monitor/:hostname", monitor.GetAgentInfo)
		auth.GET("/monitor/status/:hostname", monitor.GetLatestSystemInfo)
	}
	router.POST("/agent/addSystem_info", monitor.ReceiveAndStoreSystemMetrics)
	router.Run("0.0.0.0:8080")
}
