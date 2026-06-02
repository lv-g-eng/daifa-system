package config

import (
	"distribution-system/models"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// getEnv 读取环境变量，为空时回退默认值（环境变量优先）
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt 读取整型环境变量，解析失败时回退默认值
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// normalizePort 保证端口带前导冒号，兼容 "8080" 与 ":8080" 两种写法
func normalizePort(p string) string {
	if !strings.HasPrefix(p, ":") {
		return ":" + p
	}
	return p
}

// 数据库配置（环境变量优先，回退本地默认值）
var (
	DBHost     = getEnv("DB_HOST", "127.0.0.1")
	DBPort     = getEnv("DB_PORT", "3306")
	DBUser     = getEnv("DB_USER", "root")
	DBPassword = getEnv("DB_PASSWORD", "123456")
	DBName     = getEnv("DB_NAME", "distribution_system")
)

// JWT配置（环境变量优先）
var (
	JWTSecret     = getEnv("JWT_SECRET", "distribution-system-secret-key-2024")
	JWTExpireHour = getEnvInt("JWT_EXPIRE_HOUR", 24*7) // 默认7天过期
)

// 服务与静态资源配置（环境变量优先）
var (
	ServerPort  = normalizePort(getEnv("SERVER_PORT", "8080"))
	FrontendDir = getEnv("FRONTEND_DIR", "../frontend") // 前端静态目录
	UploadDir   = getEnv("UPLOAD_DIR", "./uploads")     // 上传文件目录
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		DBUser, DBPassword, DBHost, DBPort, DBName)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("获取数据库实例失败: %v", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// 自动迁移数据库表结构
	err = DB.AutoMigrate(
		&models.User{},
		&models.Platform{},
		&models.Material{},
		&models.PublishCode{},
		&models.Task{},
		&models.TaskSubmission{},
		&models.Withdrawal{},
		&models.Order{},
		&models.Invitation{},
		&models.Commission{},
		&models.SystemConfig{},
	)
	if err != nil {
		log.Printf("数据库迁移警告: %v", err)
	} else {
		log.Println("数据库表结构同步完成!")
	}

	// 同步parent_id：从invitations表更新到users表
	// 修复有邀请记录但parent_id为空的用户
	var invitations []models.Invitation
	DB.Find(&invitations)
	syncCount := 0
	for _, inv := range invitations {
		var user models.User
		if err := DB.First(&user, inv.InviteeID).Error; err == nil {
			if user.ParentID == nil {
				user.ParentID = &inv.InviterID
				DB.Save(&user)
				syncCount++
				log.Printf("同步用户 %d 的parent_id为 %d", inv.InviteeID, inv.InviterID)
			}
		}
	}
	if syncCount > 0 {
		log.Printf("同步了 %d 个用户的parent_id", syncCount)
	}

	log.Println("数据库连接成功!")
}
