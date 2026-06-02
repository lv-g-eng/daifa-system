package handlers

import (
	"distribution-system/config"
	"distribution-system/models"
	"distribution-system/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateMerchantRequest 创建商家请求
type CreateMerchantRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Nickname   string `json:"nickname"`
	Phone      string `json:"phone"`
	ExpireDays int    `json:"expire_days"` // 有效天数（可选，默认30天）
}

// CreateMerchant 创建商家账号 (总后台)
func CreateMerchant(c *gin.Context) {
	var req CreateMerchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("CreateMerchant 参数绑定失败: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 检查用户名是否存在
	var count int64
	config.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名已存在"})
		return
	}

	// 默认30天有效期
	expireDays := req.ExpireDays
	if expireDays <= 0 {
		expireDays = 30
	}
	expireTime := time.Now().AddDate(0, 0, expireDays)

	hashedPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	merchant := models.User{
		Username:   req.Username,
		Password:   hashedPwd,
		Nickname:   req.Nickname,
		Phone:      req.Phone,
		Role:       "merchant",
		Status:     "active",
		ExpireTime: &expireTime,
	}

	if err := config.DB.Create(&merchant).Error; err != nil {
		fmt.Printf("CreateMerchant 数据库创建失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "商家账号创建成功",
		"data": gin.H{
			"id":          merchant.ID,
			"username":    merchant.Username,
			"nickname":    merchant.Nickname,
			"expire_time": merchant.ExpireTime,
		},
	})
}

// ListMerchants 商家列表 (总后台)
func ListMerchants(c *gin.Context) {
	var merchants []models.User
	config.DB.Where("role = ?", "merchant").Order("created_at DESC").Find(&merchants)

	list := make([]gin.H, 0)
	for _, m := range merchants {
		list = append(list, gin.H{
			"id":          m.ID,
			"username":    m.Username,
			"nickname":    m.Nickname,
			"phone":       m.Phone,
			"status":      m.Status,
			"expire_time": m.ExpireTime,
			"created_at":  m.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": list,
	})
}

// UpdateMerchant 更新商家信息 (总后台)
func UpdateMerchant(c *gin.Context) {
	id := c.Param("id")

	var merchant models.User
	if err := config.DB.First(&merchant, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "商家不存在"})
		return
	}

	var req struct {
		Nickname   string `json:"nickname"`
		Phone      string `json:"phone"`
		Password   string `json:"password"`
		Status     string `json:"status"`
		ExpireDays int    `json:"expire_days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Nickname != "" {
		merchant.Nickname = req.Nickname
	}
	if req.Phone != "" {
		merchant.Phone = req.Phone
	}
	if req.Password != "" {
		if hashed, err := utils.HashPassword(req.Password); err == nil {
			merchant.Password = hashed
		}
	}
	if req.Status != "" {
		merchant.Status = req.Status
	}
	if req.ExpireDays > 0 {
		expireTime := time.Now().AddDate(0, 0, req.ExpireDays)
		merchant.ExpireTime = &expireTime
	}

	config.DB.Save(&merchant)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
	})
}

// DeleteMerchant 删除商家 (总后台)
func DeleteMerchant(c *gin.Context) {
	id := c.Param("id")

	var merchant models.User
	if err := config.DB.First(&merchant, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "商家不存在"})
		return
	}

	config.DB.Delete(&merchant)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// AdminStats 总后台统计
func AdminStats(c *gin.Context) {
	var merchantCount, userCount, taskCount, submissionCount int64
	var pendingWithdrawals, pendingPayments, expiringMerchants int64
	var todayUsers, todaySubmissions int64

	config.DB.Model(&models.User{}).Where("role = ?", "merchant").Count(&merchantCount)
	config.DB.Model(&models.User{}).Where("role = ?", "user").Count(&userCount)
	config.DB.Model(&models.Task{}).Count(&taskCount)
	config.DB.Model(&models.TaskSubmission{}).Count(&submissionCount)

	// 待处理事项
	config.DB.Model(&models.Withdrawal{}).Where("status = ?", "pending").Count(&pendingWithdrawals)
	config.DB.Model(&models.User{}).Where("role = ? AND payment_qrcode IS NOT NULL AND payment_qrcode != '' AND payment_verified = ?", "user", false).Count(&pendingPayments)

	// 7天内过期的商家
	expireDate := time.Now().AddDate(0, 0, 7)
	config.DB.Model(&models.User{}).Where("role = ? AND expire_time IS NOT NULL AND expire_time < ?", "merchant", expireDate).Count(&expiringMerchants)

	// 今日数据
	today := time.Now().Format("2006-01-02")
	config.DB.Model(&models.User{}).Where("role = ? AND DATE(created_at) = ?", "user", today).Count(&todayUsers)
	config.DB.Model(&models.TaskSubmission{}).Where("DATE(created_at) = ?", today).Count(&todaySubmissions)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"merchant_count":      merchantCount,
			"user_count":          userCount,
			"task_count":          taskCount,
			"submission_count":    submissionCount,
			"pending_withdrawals": pendingWithdrawals,
			"pending_payments":    pendingPayments,
			"expiring_merchants":  expiringMerchants,
			"today_users":         todayUsers,
			"today_submissions":   todaySubmissions,
		},
	})
}

// ListUsers 用户列表 (总后台)
func ListUsers(c *gin.Context) {
	var users []models.User
	config.DB.Where("role = ?", "user").Order("created_at DESC").Find(&users)

	list := make([]gin.H, 0)
	for _, u := range users {
		list = append(list, gin.H{
			"id":               u.ID,
			"username":         u.Username,
			"nickname":         u.Nickname,
			"phone":            u.Phone,
			"balance":          u.Balance,
			"status":           u.Status,
			"payment_qrcode":   u.PaymentQRCode,
			"payment_verified": u.PaymentVerified,
			"parent_id":        u.ParentID,
			"created_at":       u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": list,
	})
}

// AdminBlockUser 禁用/启用用户 (总后台)
func AdminBlockUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	var req struct {
		Block bool `json:"block"`
	}
	c.ShouldBindJSON(&req)

	if req.Block {
		user.Status = "blocked"
	} else {
		user.Status = "active"
	}
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "操作成功",
	})
}

// AdminDeleteUser 删除用户 (总后台)
func AdminDeleteUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 不允许删除管理员和商家账号
	if user.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能删除普通用户"})
		return
	}

	config.DB.Delete(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// ListWithdrawals 提现列表 (总后台)
func ListWithdrawals(c *gin.Context) {
	status := c.Query("status")

	var withdrawals []models.Withdrawal
	query := config.DB.Preload("User").Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Find(&withdrawals)

	var list []gin.H
	for _, w := range withdrawals {
		username := ""
		nickname := ""
		if w.User != nil {
			username = w.User.Username
			nickname = w.User.Nickname
		}

		list = append(list, gin.H{
			"id":              w.ID,
			"user_id":         w.UserID,
			"user":            gin.H{"username": username, "nickname": nickname},
			"amount":          w.Amount,
			"payment_method":  w.PaymentMethod,
			"payment_account": w.PaymentAccount,
			"payment_qrcode":  w.PaymentQRCode,
			"status":          w.Status,
			"created_at":      w.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": list,
	})
}

// AdminProcessWithdrawal 处理提现申请 (总后台)
func AdminProcessWithdrawal(c *gin.Context) {
	id := c.Param("id")

	var withdrawal models.Withdrawal
	if err := config.DB.First(&withdrawal, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "提现记录不存在"})
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	c.ShouldBindJSON(&req)

	if req.Status == "rejected" && withdrawal.Status == "pending" {
		// 拒绝时退还余额
		var user models.User
		config.DB.First(&user, withdrawal.UserID)
		user.Balance += withdrawal.Amount
		config.DB.Save(&user)
	}

	withdrawal.Status = req.Status
	config.DB.Save(&withdrawal)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "处理成功",
	})
}

// ListPaymentVerifications 待验证收款码列表
func ListPaymentVerifications(c *gin.Context) {
	var users []models.User
	config.DB.Where("role = ? AND payment_qrcode IS NOT NULL AND payment_qrcode != ''", "user").
		Order("CASE WHEN payment_verified = false THEN 0 ELSE 1 END, updated_at DESC").
		Find(&users)

	var list []gin.H
	for _, u := range users {
		list = append(list, gin.H{
			"id":               u.ID,
			"username":         u.Username,
			"nickname":         u.Nickname,
			"payment_qrcode":   u.PaymentQRCode,
			"payment_verified": u.PaymentVerified,
			"updated_at":       u.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": list,
	})
}
