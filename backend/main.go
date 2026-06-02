package main

import (
	"distribution-system/config"
	"distribution-system/handlers"
	"distribution-system/middleware"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	config.InitDB()

	// 启动定时任务
	handlers.StartScheduledTasks()

	// 创建上传目录（可由 UPLOAD_DIR 环境变量覆盖）
	os.MkdirAll(config.UploadDir, 0755)

	// 创建Gin引擎
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORSMiddleware())

	// 静态文件（前端目录可由 FRONTEND_DIR 环境变量覆盖）
	r.Static("/uploads", config.UploadDir)
	r.Static("/assets", config.FrontendDir)

	// 前端页面路由
	r.StaticFile("/", config.FrontendDir+"/index.html")
	r.StaticFile("/admin", config.FrontendDir+"/admin/index.html")
	r.StaticFile("/merchant", config.FrontendDir+"/merchant/index.html")
	r.StaticFile("/user", config.FrontendDir+"/user/index.html")

	// 发布码跳转页面
	r.GET("/p/:code", handlers.PublishCodeRedirect)

	// API路由
	api := r.Group("/api")
	{
		// 兼容一些反代场景：只转发 /api，不转发 /uploads
		api.Static("/uploads", config.UploadDir)

		// 公共接口
		api.POST("/login", handlers.Login)
		api.POST("/register", handlers.Register)
		api.GET("/platforms", handlers.ListPlatforms)
		api.GET("/publish-codes/:code", handlers.UsePublishCode)
		api.GET("/withdraw-config", handlers.GetWithdrawConfig)

		// 文件上传
		api.POST("/upload", handlers.UploadFile)
		api.POST("/upload/batch", handlers.BatchUploadFiles)

		// 需要认证的接口
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			// 用户信息
			auth.GET("/user/info", handlers.GetUserInfo)
			auth.GET("/categories", handlers.GetCategories)

			// 总后台 (admin)
			admin := auth.Group("/admin")
			admin.Use(middleware.RoleMiddleware("admin"))
			{
				admin.GET("/stats", handlers.AdminStats)
				admin.GET("/merchants", handlers.ListMerchants)
				admin.POST("/merchants", handlers.CreateMerchant)
				admin.PUT("/merchants/:id", handlers.UpdateMerchant)
				admin.DELETE("/merchants/:id", handlers.DeleteMerchant)
				admin.PUT("/users/:id/verify-payment", handlers.VerifyPaymentQRCode)

				// 用户管理
				admin.GET("/users", handlers.ListUsers)
				admin.PUT("/users/:id/block", handlers.AdminBlockUser)
				admin.DELETE("/users/:id", handlers.AdminDeleteUser)

				// 收款码审核
				admin.GET("/payments", handlers.ListPaymentVerifications)

				// 数据导出（Excel）
				admin.GET("/export/users", handlers.ExportAdminUsers)
				admin.GET("/export/merchants", handlers.ExportAdminMerchants)
				admin.GET("/export/withdrawals", handlers.ExportMerchantWithdrawals)

				// 群发站内通知
				admin.POST("/notifications/broadcast", handlers.BroadcastNotification)
			}

			// 商家后台 (merchant)
			merchant := auth.Group("/merchant")
			merchant.Use(middleware.RoleMiddleware("merchant"))
			{
				merchant.GET("/stats", handlers.MerchantStats)

				// 素材管理
				merchant.GET("/materials", handlers.ListMaterials)
				merchant.GET("/materials/:id", handlers.GetMaterial)
				merchant.PUT("/materials/:id", handlers.UpdateMaterial)
				merchant.POST("/materials", handlers.CreateMaterial)
				merchant.POST("/materials/batch", handlers.BatchCreateMaterials)

				// 发布码
				merchant.GET("/publish-codes", handlers.ListPublishCodes)
				merchant.POST("/publish-codes", handlers.CreatePublishCode)

				// 任务管理
				merchant.GET("/tasks", handlers.ListMerchantTasks)
				merchant.POST("/tasks", handlers.CreateTask)

				// 审核管理
				merchant.GET("/submissions", handlers.ListSubmissions)
				merchant.PUT("/submissions/:id", handlers.ReviewSubmission)
				merchant.POST("/submissions/:id/check", handlers.CheckLinkValidity)
				merchant.POST("/link-info", handlers.FetchLinkInfo)

				// 用户管理
				merchant.GET("/users", handlers.ListMerchantUsers)
				merchant.PUT("/users/:id/block", handlers.BlockUser)

				// 提现管理
				merchant.GET("/withdrawals", handlers.ListMerchantWithdrawals)
				merchant.PUT("/withdrawals/:id", handlers.ProcessWithdrawal)

				// 订单管理
				merchant.GET("/orders", handlers.ListMerchantOrders)
				merchant.GET("/orders/stats", handlers.GetOrderStats)

				// 分销管理
				merchant.GET("/distribution/stats", handlers.GetDistributionStats)
				merchant.GET("/invitations", handlers.ListInvitations)

				// 系统设置
				merchant.GET("/config", handlers.GetMerchantConfig)
				merchant.PUT("/config", handlers.UpdateMerchantConfig)

				// 数据导出（Excel）
				merchant.GET("/export/tasks", handlers.ExportMerchantTasks)
				merchant.GET("/export/withdrawals", handlers.ExportMerchantWithdrawals)
				merchant.GET("/export/commissions", handlers.ExportMerchantCommissions)
			}

			// 用户端 (user)
			user := auth.Group("/user")
			user.Use(middleware.RoleMiddleware("user", "merchant", "admin"))
			{
				user.GET("/home", handlers.UserHome)
				user.GET("/tasks", handlers.ListUserTasks)
				user.GET("/tasks/:id", handlers.GetTaskDetail)
				user.GET("/materials", handlers.ListUserMaterials)
				user.GET("/materials/:id", handlers.GetMaterialDetail)
				user.GET("/publish-codes", handlers.ListUserPublishCodes)
				user.POST("/submissions", handlers.SubmitTask)
				user.GET("/submissions", handlers.ListUserSubmissions)
				user.GET("/profile", handlers.GetUserProfile)
				user.PUT("/profile", handlers.UpdateProfile)
				user.PUT("/password", handlers.ChangePassword)
				user.POST("/withdraw", handlers.RequestWithdraw)
				user.GET("/withdrawals", handlers.ListUserWithdrawals)
				user.GET("/orders", handlers.ListUserOrders)
				user.GET("/referrals", handlers.GetReferralInfo)

				// 消息通知中心
				user.GET("/notifications", handlers.ListNotifications)
				user.GET("/notifications/unread-count", handlers.UnreadNotificationCount)
				user.PUT("/notifications/read", handlers.MarkNotificationRead)
				user.PUT("/notifications/read-all", handlers.MarkAllNotificationsRead)
			}
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 启动服务
	log.Printf("服务启动在 http://localhost%s", config.ServerPort)
	if err := r.Run(config.ServerPort); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
