package handlers

import (
	"distribution-system/config"
	"distribution-system/middleware"
	"distribution-system/models"
	"distribution-system/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Nickname   string `json:"nickname"`
	Phone      string `json:"phone"`
	InviteCode string `json:"invite_code"`
}

// Login 用户登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var user models.User
	if err := config.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	// 校验密码（兼容历史明码，bcrypt 优先）
	if !utils.CheckPassword(user.Password, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	// 登录时自动升级：历史明码密码在校验通过后重新哈希入库
	if !utils.IsHashed(user.Password) {
		if hashed, err := utils.HashPassword(req.Password); err == nil {
			config.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("password", hashed)
		}
	}

	// 检查账号状态 - 管理员永远可以登录
	if user.Status == "blocked" && user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "系统错误，请联系客服"})
		return
	}

	// 生成JWT
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登录成功",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"nickname": user.Nickname,
				"role":     user.Role,
				"avatar":   user.Avatar,
				"balance":  user.Balance,
			},
		},
	})
}

// Register 用户注册
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 检查用户名是否存在
	var count int64
	config.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名已存在"})
		return
	}

	// 密码哈希后存储
	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "注册失败"})
		return
	}

	// 创建用户
	user := models.User{
		Username: req.Username,
		Password: hashedPwd,
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Role:     "user",
		Status:   "active",
	}

	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "注册失败"})
		return
	}

	// 处理邀请码
	if req.InviteCode != "" {
		log.Printf("[Register] Processing invite code: %s for user: %s (ID: %d)", req.InviteCode, user.Username, user.ID)
		var inviter models.User
		if err := config.DB.Where("username = ?", req.InviteCode).First(&inviter).Error; err == nil {
			log.Printf("[Register] Found inviter: %s (ID: %d)", inviter.Username, inviter.ID)
			invitation := models.Invitation{
				InviterID:  inviter.ID,
				InviteeID:  user.ID,
				InviteCode: req.InviteCode,
			}
			if err := config.DB.Create(&invitation).Error; err != nil {
				log.Printf("[Register] Failed to create invitation: %v", err)
			} else {
				log.Printf("[Register] Created invitation successfully, ID: %d", invitation.ID)
			}
			user.ParentID = &inviter.ID
			if err := config.DB.Save(&user).Error; err != nil {
				log.Printf("[Register] Failed to update user parent_id: %v", err)
			} else {
				log.Printf("[Register] Updated user parent_id to: %d", inviter.ID)
			}
		} else {
			log.Printf("[Register] Invite code '%s' not found (no user with this username)", req.InviteCode)
		}
	}

	// 生成JWT
	token, _ := middleware.GenerateToken(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "注册成功",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"nickname": user.Nickname,
				"role":     user.Role,
			},
		},
	})
}

// GetUserInfo 获取当前用户信息
func GetUserInfo(c *gin.Context) {
	user, _ := c.Get("user")
	u := user.(*models.User)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"id":               u.ID,
			"username":         u.Username,
			"nickname":         u.Nickname,
			"phone":            u.Phone,
			"avatar":           u.Avatar,
			"role":             u.Role,
			"balance":          u.Balance,
			"payment_qrcode":   u.PaymentQRCode,
			"payment_verified": u.PaymentVerified,
		},
	})
}
