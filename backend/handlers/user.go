package handlers

import (
	"distribution-system/config"
	"distribution-system/models"
	"distribution-system/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

// UserHome 用户首页
func UserHome(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	// 获取活跃任务数
	var activeTaskCount int64
	config.DB.Model(&models.Task{}).Where("status = ?", "active").Count(&activeTaskCount)

	// 获取用户提交统计
	var pendingCount, approvedCount, rejectedCount int64
	config.DB.Model(&models.TaskSubmission{}).Where("user_id = ? AND status = ?", uid, "pending").Count(&pendingCount)
	config.DB.Model(&models.TaskSubmission{}).Where("user_id = ? AND status = ?", uid, "approved").Count(&approvedCount)
	config.DB.Model(&models.TaskSubmission{}).Where("user_id = ? AND status = ?", uid, "rejected").Count(&rejectedCount)

	// 获取用户余额
	var user models.User
	config.DB.First(&user, uid)

	// 获取用户已提交的任务ID列表
	var submittedTaskIDs []uint
	config.DB.Model(&models.TaskSubmission{}).
		Where("user_id = ?", uid).
		Pluck("task_id", &submittedTaskIDs)

	// 获取最新任务（排除已提交的）
	var latestTasks []models.Task
	latestTasksQuery := config.DB.Where("status = ?", "active")
	if len(submittedTaskIDs) > 0 {
		latestTasksQuery = latestTasksQuery.Where("id NOT IN ?", submittedTaskIDs)
	}
	latestTasksQuery.Preload("Platform").
		Order("created_at DESC").
		Limit(5).
		Find(&latestTasks)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"task_count":     activeTaskCount,
			"pending_count":  pendingCount,
			"approved_count": approvedCount,
			"rejected_count": rejectedCount,
			"balance":        user.Balance,
			"latest_tasks":   latestTasks,
		},
	})
}

// ListUserTasks 任务列表 (用户)
func ListUserTasks(c *gin.Context) {
	platformID := c.Query("platform_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 获取当前用户ID
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	// 获取用户已提交的任务ID列表（排除已提交的任务）
	var submittedTaskIDs []uint
	config.DB.Model(&models.TaskSubmission{}).
		Where("user_id = ?", uid).
		Pluck("task_id", &submittedTaskIDs)

	var tasks []models.Task
	var total int64

	query := config.DB.Model(&models.Task{}).Where("status = ?", "active")
	if platformID != "" {
		query = query.Where("platform_id = ?", platformID)
	}

	// 排除用户已提交的任务
	if len(submittedTaskIDs) > 0 {
		query = query.Where("id NOT IN ?", submittedTaskIDs)
	}

	query.Count(&total)
	query.Preload("Platform").Preload("Merchant").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tasks)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":      tasks,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetTaskDetail 任务详情
func GetTaskDetail(c *gin.Context) {
	id := c.Param("id")

	var task models.Task
	if err := config.DB.Preload("Platform").Preload("Merchant").First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": task,
	})
}

// ListUserMaterials 内容下载 (用户)
func ListUserMaterials(c *gin.Context) {
	platformID := c.Query("platform_id")
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var materials []models.Material
	var total int64

	query := config.DB.Model(&models.Material{}).Where("status = ?", "active")
	if platformID != "" {
		query = query.Where("platform_id = ?", platformID)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	query.Count(&total)
	query.Preload("Platform").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&materials)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":      materials,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetMaterialDetail 素材详情
func GetMaterialDetail(c *gin.Context) {
	id := c.Param("id")

	var material models.Material
	if err := config.DB.Preload("Platform").First(&material, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "素材不存在"})
		return
	}

	// 增加下载次数
	config.DB.Model(&material).Update("download_count", material.DownloadCount+1)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": material,
	})
}

// UsePublishCode 扫码使用发布码
func UsePublishCode(c *gin.Context) {
	code := c.Param("code")

	var publishCode models.PublishCode
	if err := config.DB.Preload("Material").Preload("Platform").
		Where("code = ?", code).First(&publishCode).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布码不存在"})
		return
	}

	// 检查状态
	if publishCode.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "发布码已失效"})
		return
	}

	// 检查扫码次数
	if publishCode.ScanCount >= publishCode.MaxScans {
		publishCode.Status = "used"
		config.DB.Save(&publishCode)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "发布码已达到使用上限"})
		return
	}

	// 检查过期
	if publishCode.ExpireTime != nil && publishCode.ExpireTime.Before(time.Now()) {
		publishCode.Status = "expired"
		config.DB.Save(&publishCode)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "发布码已过期"})
		return
	}

	// 更新扫码次数
	publishCode.ScanCount++
	if publishCode.ScanCount >= publishCode.MaxScans {
		publishCode.Status = "used"
	}
	config.DB.Save(&publishCode)

	publishURL := ""
	if publishCode.Platform != nil {
		publishURL = publishCode.Platform.PublishURL
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "验证成功",
		"data": gin.H{
			"material":    publishCode.Material,
			"platform":    publishCode.Platform,
			"publish_url": publishURL,
		},
	})
}

// ListUserPublishCodes 用户可用发布码列表
func ListUserPublishCodes(c *gin.Context) {
	platformID := c.Query("platform_id")
	now := time.Now()

	var codes []models.PublishCode
	query := config.DB.Model(&models.PublishCode{}).
		Preload("Material").
		Preload("Platform").
		Where("status = ?", "active").
		Where("(expire_time IS NULL OR expire_time > ?)", now).
		Where("scan_count < max_scans")

	if platformID != "" {
		query = query.Where("platform_id = ?", platformID)
	}

	query.Order("created_at DESC").Find(&codes)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": codes,
	})
}

// SubmitTaskRequest 提交任务请求
type SubmitTaskRequest struct {
	TaskID        uint   `json:"task_id" binding:"required"`
	MaterialNo    string `json:"material_no"`
	PublishLink   string `json:"publish_link" binding:"required"`
	ScreenshotURL string `json:"screenshot_url"`
}

// SubmitTask 链接回传
func SubmitTask(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var req SubmitTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 检查任务是否存在
	var task models.Task
	if err := config.DB.First(&task, req.TaskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}

	// 检查是否已提交
	var existCount int64
	config.DB.Model(&models.TaskSubmission{}).
		Where("task_id = ? AND user_id = ? AND publish_link = ?", req.TaskID, uid, req.PublishLink).
		Count(&existCount)
	if existCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该链接已提交过"})
		return
	}

	// 确定提交状态（自动审核逻辑）
	status := "pending"
	message := "提交成功，等待审核"

	if task.AutoReview {
		// 开启了自动审核
		if task.SampleRate <= 0 {
			// 没有抽查，全部自动通过
			status = "approved"
			message = "提交成功，已自动审核通过"
		} else {
			// 根据抽查比例决定是否需要人工审核
			// 使用随机数决定是否需要抽查
			rand.Seed(time.Now().UnixNano())
			if rand.Intn(100) < task.SampleRate {
				// 被抽查，需要人工审核
				status = "pending"
				message = "提交成功，等待抽查审核"
			} else {
				// 不抽查，自动通过
				status = "approved"
				message = "提交成功，已自动审核通过"
			}
		}
	}

	submission := models.TaskSubmission{
		TaskID:        req.TaskID,
		UserID:        uid,
		MaterialNo:    req.MaterialNo,
		PublishLink:   req.PublishLink,
		ScreenshotURL: req.ScreenshotURL,
		Status:        status,
		LinkValid:     true,
	}

	if err := config.DB.Create(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "提交失败"})
		return
	}

	// 如果自动审核通过，更新用户余额和佣金
	if status == "approved" {
		// 设置提交金额信息
		submission.UnitPrice = task.UnitPrice
		submission.TeamLeaderAmount = task.TeamLeaderPrice
		submission.BonusAmount = task.BonusPrice
		submission.TotalAmount = task.UnitPrice + task.TeamLeaderPrice + task.BonusPrice

		// 设置有效期
		validityExpire := time.Now().AddDate(0, 0, task.ValidityDays)
		submission.ValidityExpire = &validityExpire
		config.DB.Save(&submission)

		// 获取用户信息
		var user models.User
		if config.DB.First(&user, uid).Error == nil {
			// 增加余额 (使用total_amount而不只是unit_price)
			user.Balance += submission.TotalAmount
			config.DB.Save(&user)

			// 更新任务完成数
			task.CompletedCount++
			config.DB.Save(&task)

			// 处理分销佣金 - 如果用户有上级
			if user.ParentID != nil {
				var cfg models.SystemConfig
				config.DB.Where("config_key = ?", "commission_rate").First(&cfg)
				rate := 10.0 // 默认10%
				if cfg.ConfigValue != "" {
					fmt.Sscanf(cfg.ConfigValue, "%f", &rate)
				}

				commission := models.Commission{
					UserID:       *user.ParentID,
					FromUserID:   user.ID,
					SubmissionID: &submission.ID,
					Amount:       submission.TotalAmount * rate / 100,
					Rate:         rate,
					Status:       "settled",
				}
				if err := config.DB.Create(&commission).Error; err == nil {
					// 将佣金加入邀请人余额
					var parent models.User
					if config.DB.First(&parent, *user.ParentID).Error == nil {
						parent.Balance += commission.Amount
						config.DB.Save(&parent)
						log.Printf("[SubmitTask] 佣金已计入邀请人余额: ParentID=%d, Amount=%.2f", *user.ParentID, commission.Amount)
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": message,
		"data":    submission,
	})
}

// ListUserSubmissions 我的提交
func ListUserSubmissions(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var submissions []models.TaskSubmission
	var total int64

	query := config.DB.Model(&models.TaskSubmission{}).Where("user_id = ?", uid)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Preload("Task").Preload("Task.Platform").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&submissions)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":      submissions,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetUserProfile 个人中心
func GetUserProfile(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var user models.User
	config.DB.First(&user, uid)

	// 统计数据
	var totalEarnings, pendingWithdraw float64
	var submissionCount, approvedCount int64

	config.DB.Model(&models.TaskSubmission{}).Where("user_id = ?", uid).Count(&submissionCount)
	config.DB.Model(&models.TaskSubmission{}).Where("user_id = ? AND status = ?", uid, "approved").Count(&approvedCount)
	config.DB.Model(&models.TaskSubmission{}).
		Where("user_id = ? AND status = ?", uid, "approved").
		Select("COALESCE(SUM(total_amount), 0)").Scan(&totalEarnings)
	config.DB.Model(&models.Withdrawal{}).
		Where("user_id = ? AND status IN ?", uid, []string{"pending", "approved"}).
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingWithdraw)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"user": gin.H{
				"id":               user.ID,
				"username":         user.Username,
				"nickname":         user.Nickname,
				"phone":            user.Phone,
				"avatar":           user.Avatar,
				"balance":          user.Balance,
				"payment_qrcode":   user.PaymentQRCode,
				"payment_verified": user.PaymentVerified,
			},
			"stats": gin.H{
				"total_earnings":   totalEarnings,
				"pending_withdraw": pendingWithdraw,
				"submission_count": submissionCount,
				"approved_count":   approvedCount,
			},
		},
	})
}

// UpdateProfile 更新个人信息
func UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var user models.User
	config.DB.First(&user, uid)

	var req struct {
		Username      string `json:"username"`
		Nickname      string `json:"nickname"`
		Phone         string `json:"phone"`
		Avatar        string `json:"avatar"`
		PaymentQRCode string `json:"payment_qrcode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Username != "" && req.Username != user.Username {
		var count int64
		config.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名已存在"})
			return
		}
		user.Username = req.Username
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.PaymentQRCode != "" {
		user.PaymentQRCode = req.PaymentQRCode
		user.PaymentVerified = false // 上传新收款码需要重新验证
	}

	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
	})
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var user models.User
	config.DB.First(&user, uid)

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 验证旧密码（兼容历史明码）
	if !utils.CheckPassword(user.Password, req.OldPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "原密码错误"})
		return
	}

	// 检查新密码长度
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "新密码至少6位"})
		return
	}

	// 更新密码（哈希存储）
	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码修改失败"})
		return
	}
	user.Password = hashed
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "密码修改成功",
	})
}

// RequestWithdraw 申请提现
func RequestWithdraw(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var user models.User
	config.DB.First(&user, uid)

	var req struct {
		Amount         float64 `json:"amount" binding:"required"`
		PaymentMethod  string  `json:"payment_method"`
		PaymentAccount string  `json:"payment_account"`
		PaymentQRCode  string  `json:"payment_qrcode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 检查最低提现金额
	var cfg models.SystemConfig
	config.DB.Where("config_key = ?", "min_withdraw_amount").First(&cfg)
	minAmount := 10.0
	if cfg.ConfigValue != "" {
		fmt.Sscanf(cfg.ConfigValue, "%f", &minAmount)
	}

	if req.Amount < minAmount {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("最低提现金额为%.2f元", minAmount)})
		return
	}

	// 检查最高提现金额
	if req.Amount > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "单笔最高提现5000元"})
		return
	}

	// 检查余额
	if user.Balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "余额不足"})
		return
	}

	// 检查收款账号
	if req.PaymentAccount == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请输入收款账号"})
		return
	}

	// 默认收款方式
	if req.PaymentMethod == "" {
		req.PaymentMethod = "alipay"
	}

	// 没传收款码时尝试使用用户已绑定的收款码
	if req.PaymentQRCode == "" && user.PaymentQRCode != "" {
		req.PaymentQRCode = user.PaymentQRCode
	}

	// 扣除余额
	user.Balance -= req.Amount
	config.DB.Save(&user)

	// 创建提现记录
	withdrawal := models.Withdrawal{
		UserID:         uid,
		Amount:         req.Amount,
		PaymentMethod:  req.PaymentMethod,
		PaymentAccount: req.PaymentAccount,
		PaymentQRCode:  req.PaymentQRCode,
		Status:         "pending",
	}
	if err := config.DB.Create(&withdrawal).Error; err != nil {
		// 回滚用户余额
		user.Balance += req.Amount
		config.DB.Save(&user)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建提现记录失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "提现申请已提交",
		"data":    withdrawal,
	})
}

// ListUserWithdrawals 提现记录
func ListUserWithdrawals(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var withdrawals []models.Withdrawal
	config.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&withdrawals)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": withdrawals,
	})
}

// ListUserOrders 订单记录
func ListUserOrders(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var orders []models.Order
	config.DB.Where("user_id = ?", uid).Order("created_at DESC").Find(&orders)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": orders,
	})
}

// GetReferralInfo 裂变代发信息
func GetReferralInfo(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var user models.User
	config.DB.First(&user, uid)

	// 邀请码就是用户名
	inviteCode := user.Username

	// 获取邀请列表
	var invitations []models.Invitation
	config.DB.Where("inviter_id = ?", uid).Preload("Invitee").Find(&invitations)

	var inviteeList []gin.H
	for _, inv := range invitations {
		if inv.Invitee != nil {
			// 计算该被邀请人贡献的佣金总额
			var contributedCommission float64
			config.DB.Model(&models.Commission{}).
				Where("user_id = ? AND from_user_id = ?", uid, inv.Invitee.ID).
				Select("COALESCE(SUM(amount), 0)").Scan(&contributedCommission)

			inviteeList = append(inviteeList, gin.H{
				"id":                     inv.Invitee.ID,
				"nickname":               inv.Invitee.Nickname,
				"created_at":             inv.CreatedAt,
				"contributed_commission": contributedCommission,
			})
		}
	}

	// 获取佣金统计
	var totalCommission, pendingCommission float64
	config.DB.Model(&models.Commission{}).
		Where("user_id = ?", uid).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalCommission)
	config.DB.Model(&models.Commission{}).
		Where("user_id = ? AND status = ?", uid, "pending").
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingCommission)

	// 获取佣金记录
	var commissions []models.Commission
	config.DB.Where("user_id = ?", uid).
		Preload("FromUser").
		Order("created_at DESC").
		Limit(20).
		Find(&commissions)

	var commissionList []gin.H
	for _, com := range commissions {
		fromNickname := ""
		if com.FromUser != nil {
			fromNickname = com.FromUser.Nickname
		}
		commissionList = append(commissionList, gin.H{
			"id":            com.ID,
			"amount":        com.Amount,
			"rate":          com.Rate,
			"from_nickname": fromNickname,
			"status":        com.Status,
			"created_at":    com.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"invite_code":        inviteCode,
			"invitee_count":      len(inviteeList),
			"invitee_list":       inviteeList,
			"total_commission":   totalCommission,
			"pending_commission": pendingCommission,
			"commission_list":    commissionList,
		},
	})
}

// UploadFile 文件上传
func UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择文件"})
		return
	}

	// 生成文件名
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
	filepath := fmt.Sprintf("uploads/%s", filename)

	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "上传失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"url":      "/" + filepath,
			"filename": filename,
		},
	})
}

// BatchUploadFiles 批量上传
func BatchUploadFiles(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择文件"})
		return
	}

	files := form.File["files"]
	var urls []string

	for _, file := range files {
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
		filepath := fmt.Sprintf("uploads/%s", filename)

		if err := c.SaveUploadedFile(file, filepath); err != nil {
			continue
		}
		urls = append(urls, "/"+filepath)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"urls": urls,
		},
	})
}

// VerifyPaymentQRCode 验证用户收款码 (管理员)
func VerifyPaymentQRCode(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := config.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	var req struct {
		Verified bool `json:"verified"`
	}
	c.ShouldBindJSON(&req)

	user.PaymentVerified = req.Verified
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "操作成功",
	})
}

// GetCategories 获取素材分类
func GetCategories(c *gin.Context) {
	var categories []string
	config.DB.Model(&models.Material{}).
		Distinct("category").
		Where("category IS NOT NULL AND category != ''").
		Pluck("category", &categories)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": categories,
	})
}

// BatchCreateMaterials 批量上传素材
func BatchCreateMaterials(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var req struct {
		Materials []struct {
			MaterialNo string   `json:"material_no"`
			PlatformID uint     `json:"platform_id"`
			Category   string   `json:"category"`
			SequenceNo int      `json:"sequence_no"`
			Title      string   `json:"title"`
			Content    string   `json:"content"`
			Topics     string   `json:"topics"`
			Images     []string `json:"images"`
			Videos     []string `json:"videos"`
			Audios     []string `json:"audios"`
		} `json:"materials"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var created int
	for _, m := range req.Materials {
		imagesJSON, _ := json.Marshal(m.Images)
		videosJSON, _ := json.Marshal(m.Videos)
		audiosJSON, _ := json.Marshal(m.Audios)

		material := models.Material{
			MerchantID: merchantID,
			MaterialNo: m.MaterialNo,
			PlatformID: &m.PlatformID,
			Category:   m.Category,
			SequenceNo: m.SequenceNo,
			Title:      m.Title,
			Content:    m.Content,
			Topics:     m.Topics,
			Images:     string(imagesJSON),
			Videos:     string(videosJSON),
			Audios:     string(audiosJSON),
			Status:     "active",
		}

		if err := config.DB.Create(&material).Error; err == nil {
			created++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": fmt.Sprintf("成功上传 %d 个素材", created),
	})
}

// PublishCodeRedirect 发布码跳转页面
func PublishCodeRedirect(c *gin.Context) {
	code := c.Param("code")
	preview := c.Query("preview") == "1" || c.Query("preview") == "true"

	var publishCode models.PublishCode
	if err := config.DB.Preload("Material").Preload("Platform").
		Where("code = ?", code).First(&publishCode).Error; err != nil {
		c.HTML(http.StatusNotFound, "", `
<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>发布码无效</title>
<style>body{font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f5f5f5;}
.error{text-align:center;padding:2rem;background:white;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,0.1);}
h1{color:#ef4444;margin-bottom:1rem;}p{color:#666;}</style></head>
<body><div class="error"><h1>❌ 发布码无效</h1><p>该发布码不存在或已失效</p></div></body></html>`)
		return
	}

	// 检查状态
	if publishCode.Status != "active" {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(`
<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>发布码已失效</title>
<style>body{font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f5f5f5;}
.error{text-align:center;padding:2rem;background:white;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,0.1);}
h1{color:#f59e0b;margin-bottom:1rem;}p{color:#666;}</style></head>
<body><div class="error"><h1>⚠️ 发布码已失效</h1><p>该发布码已使用或已过期</p></div></body></html>`))
		return
	}

	// 检查扫码次数和过期
	if publishCode.ScanCount >= publishCode.MaxScans {
		if !preview {
			publishCode.Status = "used"
			config.DB.Save(&publishCode)
		}
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(`
<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>发布码已用尽</title>
<style>body{font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f5f5f5;}
.error{text-align:center;padding:2rem;background:white;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,0.1);}
h1{color:#f59e0b;margin-bottom:1rem;}p{color:#666;}</style></head>
<body><div class="error"><h1>⚠️ 发布码已用尽</h1><p>该发布码已达到使用次数上限</p></div></body></html>`))
		return
	}

	if publishCode.ExpireTime != nil && publishCode.ExpireTime.Before(time.Now()) {
		if !preview {
			publishCode.Status = "expired"
			config.DB.Save(&publishCode)
		}
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(`
<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>发布码已过期</title>
<style>body{font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f5f5f5;}
.error{text-align:center;padding:2rem;background:white;border-radius:12px;box-shadow:0 2px 10px rgba(0,0,0,0.1);}
h1{color:#f59e0b;margin-bottom:1rem;}p{color:#666;}</style></head>
<body><div class="error"><h1>⚠️ 发布码已过期</h1><p>该发布码已超过有效期</p></div></body></html>`))
		return
	}

	// 更新扫码次数（预览模式不消耗次数）
	if !preview {
		publishCode.ScanCount++
		if publishCode.ScanCount >= publishCode.MaxScans {
			publishCode.Status = "used"
		}
		config.DB.Save(&publishCode)
	}

	// 获取发布链接
	publishURL := "https://creator.xiaohongshu.com/"
	platformName := "小红书"
	if publishCode.Platform != nil {
		platformName = publishCode.Platform.Name
		if publishCode.Platform.PublishURL != "" {
			publishURL = publishCode.Platform.PublishURL
		} else if publishCode.Platform.Code == "douyin" {
			publishURL = "https://creator.douyin.com/creator-micro/content/upload"
		} else if publishCode.Platform.Code == "kuaishou" {
			publishURL = "https://cp.kuaishou.com/article/publish/video"
		} else if publishCode.Platform.Code == "weibo" {
			publishURL = "https://weibo.com/compose/"
		}
	}

	platformIcon := "?"
	if platformName != "" {
		runes := []rune(platformName)
		if len(runes) > 0 {
			platformIcon = string(runes[0])
		}
	}

	qrBase64 := ""
	if png, err := qrcode.Encode(publishURL, qrcode.Medium, 220); err == nil {
		qrBase64 = base64.StdEncoding.EncodeToString(png)
	}

	// URL Scheme for native apps
	appScheme := ""
	if publishCode.Platform != nil {
		switch publishCode.Platform.Code {
		case "xiaohongshu":
			appScheme = "xhsdiscover://post"
		case "douyin":
			appScheme = "snssdk1128://"
		case "kuaishou":
			appScheme = "kwai://"
		case "weibo":
			appScheme = "sinaweibo://"
		}
	}

	platformBlock := fmt.Sprintf(`
			<div class="platform">
				<div class="platform-icon">%s</div>
				<div class="platform-name">%s</div>
			</div>`, platformIcon, platformName)
	if qrBase64 != "" {
		platformBlock = fmt.Sprintf(`
			<div class="platform">
				<div class="platform-qr">
					<img src="data:image/png;base64,%s" alt="%s二维码">
				</div>
				<div class="platform-info">
					<div class="platform-name">%s</div>
					<div class="platform-tip">微信扫码跳转到%s</div>
				</div>
			</div>`, qrBase64, platformName, platformName, platformName)
	}

	// 构建素材信息HTML（带复制功能）
	materialHTML := ""
	if publishCode.Material != nil {
		contentEscaped := template.HTMLEscapeString(publishCode.Material.Content)
		topicsEscaped := template.HTMLEscapeString(publishCode.Material.Topics)
		materialHTML = fmt.Sprintf(`
<div class="material">
	<h3>📄 素材内容</h3>
	<div class="material-item">
		<div class="material-label">标题</div>
		<div class="material-content" id="titleText">%s</div>
		<button class="copy-btn" onclick="copyText('titleText', this)">📋 复制</button>
	</div>
	<div class="material-item">
		<div class="material-label">正文</div>
		<div class="material-content" id="contentText">%s</div>
		<button class="copy-btn" onclick="copyText('contentText', this)">📋 复制</button>
	</div>
	<div class="material-item">
		<div class="material-label">话题</div>
		<div class="material-content" id="topicsText">%s</div>
		<button class="copy-btn" onclick="copyText('topicsText', this)">📋 复制</button>
	</div>
</div>`, publishCode.Material.Title, contentEscaped, topicsEscaped)
	}

	// App按钮HTML
	appButtonHTML := ""
	if appScheme != "" {
		appButtonHTML = fmt.Sprintf(`
			<button class="btn btn-app" data-app="%s" onclick="oneClickFill(this)">⚡ 一键填充并打开APP</button>
		`, appScheme)
	}

	// 返回跳转页面HTML
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
	<title>发布码 - %s</title>
	<style>
		* { box-sizing: border-box; }
		body { font-family: system-ui, -apple-system, sans-serif; margin: 0; padding: 16px; 
			background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); min-height: 100vh; }
		.container { max-width: 480px; margin: 0 auto; background: white; border-radius: 20px;
			box-shadow: 0 20px 60px rgba(0,0,0,0.3); overflow: hidden; }
		.header { background: linear-gradient(135deg, #6366f1 0%%, #8b5cf6 100%%); color: white;
			padding: 28px 20px; text-align: center; }
		.header h1 { margin: 0 0 8px 0; font-size: 1.5rem; }
		.header p { margin: 0; opacity: 0.9; font-size: 0.9rem; }
		.content { padding: 20px; }
		.platform { display: flex; align-items: center; gap: 16px; padding: 20px;
			background: #f8fafc; border-radius: 16px; margin-bottom: 16px; }
		.platform-icon { width: 56px; height: 56px; background: linear-gradient(135deg, #ef4444, #f97316); color: white;
			border-radius: 14px; display: flex; align-items: center; justify-content: center;
			font-weight: bold; font-size: 1.5rem; }
		.platform-qr { width: 120px; height: 120px; padding: 8px; background: white; border-radius: 12px;
			border: 2px solid #e5e7eb; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
		.platform-qr img { width: 100%%; height: 100%%; object-fit: contain; }
		.platform-info { display: flex; flex-direction: column; gap: 6px; }
		.platform-name { font-weight: 700; font-size: 1.2rem; color: #1e293b; }
		.platform-tip { font-size: 0.85rem; color: #64748b; }
		.material { padding: 16px; background: #f0fdf4; border-radius: 16px; margin-bottom: 16px;
			border: 1px solid #bbf7d0; }
		.material h3 { margin: 0 0 16px 0; font-size: 1rem; color: #166534; }
		.material-item { margin-bottom: 12px; position: relative; }
		.material-label { font-size: 0.75rem; color: #6b7280; margin-bottom: 4px; text-transform: uppercase; font-weight: 600; }
		.material-content { font-size: 0.9rem; color: #1f2937; padding: 12px; background: white;
			border-radius: 10px; word-break: break-all; max-height: 150px; overflow-y: auto; }
		.copy-btn { position: absolute; top: 0; right: 0; background: #10b981; color: white; border: none;
			padding: 6px 12px; border-radius: 6px; font-size: 0.75rem; cursor: pointer; transition: all 0.2s; }
		.copy-btn:hover { background: #059669; transform: scale(1.05); }
		.copy-btn.copied { background: #6366f1; }
		.btn-group { display: flex; flex-direction: column; gap: 12px; margin-top: 20px; }
		.btn { display: flex; align-items: center; justify-content: center; gap: 8px; width: 100%%; padding: 16px;
			color: white; text-align: center; text-decoration: none; border-radius: 14px;
			font-weight: 600; font-size: 1rem; transition: transform 0.2s, box-shadow 0.2s; border: none; cursor: pointer; }
		.btn-primary { background: linear-gradient(135deg, #6366f1 0%%, #8b5cf6 100%%); }
		.btn-app { background: linear-gradient(135deg, #10b981 0%%, #059669 100%%); }
		.btn:hover { transform: translateY(-2px); box-shadow: 0 8px 20px rgba(99, 102, 241, 0.4); }
		.note { margin-top: 16px; padding: 14px; background: linear-gradient(135deg, #fef3c7, #fde68a); border-radius: 12px;
			font-size: 0.85rem; color: #92400e; text-align: center; }
		@media (max-width: 480px) {
			body { padding: 12px; }
			.platform { flex-direction: column; align-items: center; text-align: center; padding: 16px; }
			.platform-qr { width: 140px; height: 140px; }
			.header { padding: 24px 16px; }
			.content { padding: 16px; }
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>✨ 发布码验证成功</h1>
			<p>请点击下方按钮前往发布</p>
		</div>
		<div class="content">
			%s
			%s
			<div class="btn-group">
				%s
				<a href="%s" target="_blank" class="btn btn-primary">🚀 前往网页发布</a>
			</div>
			<div class="note">💡 请确保已登录对应平台账号%s</div>
		</div>
	</div>
	<script>
		function copyText(elementId, btn) {
			const text = document.getElementById(elementId).innerText;
			if (navigator.clipboard) {
				navigator.clipboard.writeText(text).then(() => {
					btn.textContent = '✅ 已复制';
					btn.classList.add('copied');
					setTimeout(() => {
						btn.textContent = '📋 复制';
						btn.classList.remove('copied');
					}, 2000);
				});
			} else {
				const textarea = document.createElement('textarea');
				textarea.value = text;
				document.body.appendChild(textarea);
				textarea.select();
				document.execCommand('copy');
				document.body.removeChild(textarea);
				btn.textContent = '✅ 已复制';
				btn.classList.add('copied');
				setTimeout(() => {
					btn.textContent = '📋 复制';
					btn.classList.remove('copied');
				}, 2000);
			}
		}
		function buildAllText() {
			const title = (document.getElementById('titleText')?.innerText || '').trim();
			const content = (document.getElementById('contentText')?.innerText || '').trim();
			const topics = (document.getElementById('topicsText')?.innerText || '').trim();
			return [title, content, topics].filter(Boolean).join('\n\n');
		}

		function buildAppUrl(appSchemeBase) {
			return appSchemeBase || '';
		}

		function copyToClipboard(text) {
			if (!text) return Promise.resolve(false);
			if (navigator.clipboard) {
				return navigator.clipboard.writeText(text).then(() => true).catch(() => fallbackCopy(text));
			}
			return fallbackCopy(text);
		}

		function fallbackCopy(text) {
			return new Promise(resolve => {
				const textarea = document.createElement('textarea');
				textarea.value = text;
				document.body.appendChild(textarea);
				textarea.select();
				document.execCommand('copy');
				document.body.removeChild(textarea);
				resolve(true);
			});
		}

		function oneClickFill(btn) {
			const appSchemeBase = btn?.dataset?.app || '';
			const text = buildAllText();
			const openTarget = () => openApp(appSchemeBase);
			if (!text) {
				openTarget();
				return;
			}
			copyToClipboard(text).finally(openTarget);
			if (btn) {
				const original = btn.textContent;
				btn.textContent = '✅ 已复制并尝试打开APP';
				setTimeout(() => { btn.textContent = original; }, 2000);
			}
		}

		function openApp(appSchemeBase) {
			// 只打开App，不打开网页版
			const appUrl = buildAppUrl(appSchemeBase);
			if (appUrl) {
				window.location.href = appUrl;
			}
		}
	</script>
</body>
</html>`, platformName, platformBlock, materialHTML, appButtonHTML, publishURL, func() string {
		if preview {
			return "（预览模式不消耗次数）"
		}
		return ""
	}())

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// GetWithdrawConfig 获取提现配置（公开接口）
func GetWithdrawConfig(c *gin.Context) {
	var minWithdrawCfg, withdrawFeeCfg, maxWithdrawCfg models.SystemConfig

	// 获取最低提现金额
	config.DB.Where("config_key = ?", "min_withdraw").First(&minWithdrawCfg)
	minWithdraw := 10.0
	if minWithdrawCfg.ConfigValue != "" {
		fmt.Sscanf(minWithdrawCfg.ConfigValue, "%f", &minWithdraw)
	}

	// 获取提现手续费
	config.DB.Where("config_key = ?", "withdraw_fee").First(&withdrawFeeCfg)
	withdrawFee := 0.0
	if withdrawFeeCfg.ConfigValue != "" {
		fmt.Sscanf(withdrawFeeCfg.ConfigValue, "%f", &withdrawFee)
	}

	// 获取最高提现金额
	config.DB.Where("config_key = ?", "max_withdraw").First(&maxWithdrawCfg)
	maxWithdraw := 5000.0
	if maxWithdrawCfg.ConfigValue != "" {
		fmt.Sscanf(maxWithdrawCfg.ConfigValue, "%f", &maxWithdraw)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"min_withdraw": minWithdraw,
			"max_withdraw": maxWithdraw,
			"withdraw_fee": withdrawFee,
			"process_days": "1-3",
		},
	})
}
