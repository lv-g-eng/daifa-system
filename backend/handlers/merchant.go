package handlers

import (
	"distribution-system/config"
	"distribution-system/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// MerchantStats 商家统计
func MerchantStats(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var taskCount, submissionCount, approvedCount, pendingCount, userCount int64
	var totalEarnings float64

	config.DB.Model(&models.Task{}).Where("merchant_id = ?", merchantID).Count(&taskCount)
	config.DB.Model(&models.TaskSubmission{}).
		Joins("JOIN tasks ON tasks.id = task_submissions.task_id").
		Where("tasks.merchant_id = ?", merchantID).Count(&submissionCount)
	config.DB.Model(&models.TaskSubmission{}).
		Joins("JOIN tasks ON tasks.id = task_submissions.task_id").
		Where("tasks.merchant_id = ? AND task_submissions.status = ?", merchantID, "approved").Count(&approvedCount)
	config.DB.Model(&models.TaskSubmission{}).
		Joins("JOIN tasks ON tasks.id = task_submissions.task_id").
		Where("tasks.merchant_id = ? AND task_submissions.status = ?", merchantID, "pending").Count(&pendingCount)
	config.DB.Model(&models.User{}).Where("role = ?", "user").Count(&userCount)
	config.DB.Model(&models.TaskSubmission{}).
		Joins("JOIN tasks ON tasks.id = task_submissions.task_id").
		Where("tasks.merchant_id = ? AND task_submissions.status = ?", merchantID, "approved").
		Select("COALESCE(SUM(task_submissions.total_amount), 0)").Scan(&totalEarnings)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"task_count":       taskCount,
			"submission_count": submissionCount,
			"approved_count":   approvedCount,
			"pending_count":    pendingCount,
			"user_count":       userCount,
			"total_earnings":   totalEarnings,
		},
	})
}

// CreateMaterial 上传素材（支持自动覆盖）
func CreateMaterial(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var req struct {
		MaterialNo string   `json:"material_no" binding:"required"`
		PlatformID uint     `json:"platform_id"`
		Category   string   `json:"category"`
		SequenceNo int      `json:"sequence_no"`
		Title      string   `json:"title" binding:"required"`
		Content    string   `json:"content"`
		Topics     string   `json:"topics"`
		Images     []string `json:"images"`
		Videos     []string `json:"videos"`
		Audios     []string `json:"audios"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	imagesJSON, _ := json.Marshal(req.Images)
	videosJSON, _ := json.Marshal(req.Videos)
	audiosJSON, _ := json.Marshal(req.Audios)

	// 检查是否存在相同素材编号的素材（同一商家）
	var existingMaterial models.Material
	result := config.DB.Where("material_no = ? AND merchant_id = ?", req.MaterialNo, merchantID).First(&existingMaterial)

	if result.Error == nil {
		// 素材已存在，执行更新（覆盖）
		existingMaterial.PlatformID = &req.PlatformID
		existingMaterial.Category = req.Category
		existingMaterial.SequenceNo = req.SequenceNo
		existingMaterial.Title = req.Title
		existingMaterial.Content = req.Content
		existingMaterial.Topics = req.Topics
		existingMaterial.Images = string(imagesJSON)
		existingMaterial.Videos = string(videosJSON)
		existingMaterial.Audios = string(audiosJSON)
		existingMaterial.Status = "active"

		if err := config.DB.Save(&existingMaterial).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "素材已更新（覆盖）",
			"data":    existingMaterial,
		})
		return
	}

	// 素材不存在，创建新素材
	material := models.Material{
		MerchantID: merchantID,
		MaterialNo: req.MaterialNo,
		PlatformID: &req.PlatformID,
		Category:   req.Category,
		SequenceNo: req.SequenceNo,
		Title:      req.Title,
		Content:    req.Content,
		Topics:     req.Topics,
		Images:     string(imagesJSON),
		Videos:     string(videosJSON),
		Audios:     string(audiosJSON),
		Status:     "active",
	}

	if err := config.DB.Create(&material).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "素材上传成功",
		"data":    material,
	})
}

// ListMaterials 素材列表
func ListMaterials(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	category := c.Query("category")

	var materials []models.Material
	var total int64

	query := config.DB.Model(&models.Material{}).Where("merchant_id = ?", merchantID)
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

// GetMaterial 获取单个素材
func GetMaterial(c *gin.Context) {
	id := c.Param("id")

	var material models.Material
	if err := config.DB.Preload("Platform").First(&material, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "素材不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": material,
	})
}

// UpdateMaterial 更新素材
func UpdateMaterial(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)
	id := c.Param("id")

	var material models.Material
	if err := config.DB.First(&material, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "素材不存在"})
		return
	}

	// 验证归属权
	if material.MerchantID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限修改此素材"})
		return
	}

	var req struct {
		PlatformID *uint    `json:"platform_id"`
		Category   string   `json:"category"`
		SequenceNo int      `json:"sequence_no"`
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		Topics     string   `json:"topics"`
		Images     []string `json:"images"`
		Videos     []string `json:"videos"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 更新字段
	if req.Title != "" {
		material.Title = req.Title
	}
	if req.Content != "" {
		material.Content = req.Content
	}
	material.Topics = req.Topics
	material.Category = req.Category
	material.SequenceNo = req.SequenceNo
	if req.PlatformID != nil {
		material.PlatformID = req.PlatformID
	}
	if req.Images != nil {
		imagesJSON, _ := json.Marshal(req.Images)
		material.Images = string(imagesJSON)
	}
	if req.Videos != nil {
		videosJSON, _ := json.Marshal(req.Videos)
		material.Videos = string(videosJSON)
	}

	if err := config.DB.Save(&material).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	// 重新加载带关联信息的素材
	config.DB.Preload("Platform").First(&material, material.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "素材更新成功",
		"data":    material,
	})
}

// CreatePublishCode 生成发布码
func CreatePublishCode(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var req struct {
		MaterialID  uint   `json:"material_id"`
		PlatformID  uint   `json:"platform_id" binding:"required"`
		ProjectName string `json:"project_name"`
		MaxScans    int    `json:"max_scans"`
		ExpireDays  int    `json:"expire_days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 生成唯一码
	code := fmt.Sprintf("PB%d%d", time.Now().UnixNano(), merchantID)

	var expireTime *time.Time
	if req.ExpireDays > 0 {
		t := time.Now().AddDate(0, 0, req.ExpireDays)
		expireTime = &t
	}

	maxScans := 1
	if req.MaxScans > 0 {
		maxScans = req.MaxScans
	}

	// MaterialID为0时设为nil，避免外键约束问题
	var materialID *uint
	if req.MaterialID > 0 {
		materialID = &req.MaterialID
	}

	publishCode := models.PublishCode{
		MerchantID:  merchantID,
		MaterialID:  materialID,
		PlatformID:  req.PlatformID,
		ProjectName: req.ProjectName,
		Code:        code,
		MaxScans:    maxScans,
		ExpireTime:  expireTime,
		Status:      "active",
	}

	if err := config.DB.Create(&publishCode).Error; err != nil {
		fmt.Printf("创建发布码失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}

	config.DB.Preload("Material").Preload("Platform").First(&publishCode, publishCode.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "发布码生成成功",
		"data":    publishCode,
	})
}

// ListPublishCodes 发布码列表
func ListPublishCodes(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var codes []models.PublishCode
	config.DB.Where("merchant_id = ?", merchantID).
		Preload("Material").
		Preload("Platform").
		Order("created_at DESC").
		Find(&codes)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": codes,
	})
}

// CreateTask 发布任务
func CreateTask(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var req struct {
		PlatformID      uint     `json:"platform_id" binding:"required"`
		ProjectName     string   `json:"project_name" binding:"required"`
		TeamGroup       string   `json:"team_group"`
		Description     string   `json:"description"`
		ExampleImages   []string `json:"example_images"`
		UnitPrice       float64  `json:"unit_price" binding:"required"`
		TeamLeaderPrice float64  `json:"team_leader_price"`
		BonusPrice      float64  `json:"bonus_price"`
		DeadlineDays    int      `json:"deadline_days"`
		ValidityDays    int      `json:"validity_days"`
		TotalCount      int      `json:"total_count"`
		AutoReview      bool     `json:"auto_review"`
		SampleRate      int      `json:"sample_rate"`
		AutoReviewDays  int      `json:"auto_review_days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	exampleImagesJSON, _ := json.Marshal(req.ExampleImages)

	var deadline *time.Time
	if req.DeadlineDays > 0 {
		t := time.Now().AddDate(0, 0, req.DeadlineDays)
		deadline = &t
	}

	validityDays := 30
	if req.ValidityDays > 0 {
		validityDays = req.ValidityDays
	}

	// 自动审核天数限制在1-30天之间
	autoReviewDays := 7 // 默认7天
	if req.AutoReviewDays > 0 && req.AutoReviewDays <= 30 {
		autoReviewDays = req.AutoReviewDays
	}

	task := models.Task{
		MerchantID:      merchantID,
		PlatformID:      req.PlatformID,
		ProjectName:     req.ProjectName,
		TeamGroup:       req.TeamGroup,
		Description:     req.Description,
		ExampleImages:   string(exampleImagesJSON),
		UnitPrice:       req.UnitPrice,
		TeamLeaderPrice: req.TeamLeaderPrice,
		BonusPrice:      req.BonusPrice,
		Deadline:        deadline,
		ValidityDays:    validityDays,
		TotalCount:      req.TotalCount,
		AutoReview:      req.AutoReview,
		SampleRate:      req.SampleRate,
		AutoReviewDays:  autoReviewDays,
		Status:          "active",
	}

	if err := config.DB.Create(&task).Error; err != nil {
		log.Printf("[CreateTask] Error creating task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "任务发布成功",
		"data":    task,
	})
}

// ListTasks 任务列表 (商家)
func ListMerchantTasks(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var tasks []models.Task
	config.DB.Where("merchant_id = ?", merchantID).
		Preload("Platform").
		Order("created_at DESC").
		Find(&tasks)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": tasks,
	})
}

// ListSubmissions 审核列表 (商家)
func ListSubmissions(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var submissions []models.TaskSubmission
	var total int64

	query := config.DB.Model(&models.TaskSubmission{}).
		Joins("JOIN tasks ON tasks.id = task_submissions.task_id").
		Where("tasks.merchant_id = ?", merchantID)

	if status != "" {
		query = query.Where("task_submissions.status = ?", status)
	}

	query.Count(&total)
	query.Preload("Task").Preload("Task.Platform").Preload("User").
		Order("task_submissions.created_at DESC").
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

// ReviewSubmission 审核提交
func ReviewSubmission(c *gin.Context) {
	id := c.Param("id")

	var submission models.TaskSubmission
	if err := config.DB.Preload("Task").First(&submission, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "提交不存在"})
		return
	}

	var req struct {
		Status       string `json:"status" binding:"required"` // approved, rejected
		RejectReason string `json:"reject_reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Printf("[ReviewSubmission] Processing submission ID=%s, new status=%s", id, req.Status)

	submission.Status = req.Status

	if req.Status == "approved" {
		// 检查Task是否已预加载
		if submission.Task == nil {
			// 重新加载Task
			var task models.Task
			if err := config.DB.First(&task, submission.TaskID).Error; err != nil {
				log.Printf("[ReviewSubmission] Error loading task: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "无法加载任务信息"})
				return
			}
			submission.Task = &task
		}

		// 设置有效期
		validityDays := 30
		if submission.Task.ValidityDays > 0 {
			validityDays = submission.Task.ValidityDays
		}
		validityExpire := time.Now().AddDate(0, 0, validityDays)
		submission.ValidityExpire = &validityExpire
		submission.UnitPrice = submission.Task.UnitPrice
		submission.TeamLeaderAmount = submission.Task.TeamLeaderPrice
		submission.BonusAmount = submission.Task.BonusPrice
		submission.TotalAmount = submission.Task.UnitPrice + submission.Task.TeamLeaderPrice + submission.Task.BonusPrice

		// 更新用户余额
		var user models.User
		if err := config.DB.First(&user, submission.UserID).Error; err == nil {
			user.Balance += submission.TotalAmount
			config.DB.Save(&user)
			log.Printf("[ReviewSubmission] Updated user %d balance, added %.2f", user.ID, submission.TotalAmount)
		}

		// 更新任务完成数
		var task models.Task
		if err := config.DB.First(&task, submission.TaskID).Error; err == nil {
			task.CompletedCount++
			config.DB.Save(&task)
		}

		// Debug: 检查用户的parent_id
		fmt.Println("[ReviewSubmission] DEBUG - User ID:", user.ID, "ParentID:", user.ParentID, "TotalAmount:", submission.TotalAmount)

		// 处理分销佣金
		if user.ParentID != nil {
			log.Printf("[ReviewSubmission] User has parent, creating commission for parent %d", *user.ParentID)
			var cfg models.SystemConfig
			config.DB.Where("config_key = ?", "commission_rate").First(&cfg)
			rate := 10.0 // 默认10%
			if cfg.ConfigValue != "" {
				fmt.Sscanf(cfg.ConfigValue, "%f", &rate)
			}
			log.Printf("[ReviewSubmission] Commission rate: %.2f%%", rate)

			commission := models.Commission{
				UserID:       *user.ParentID,
				FromUserID:   user.ID,
				SubmissionID: &submission.ID,
				Amount:       submission.TotalAmount * rate / 100,
				Rate:         rate,
				Status:       "settled",
			}
			if err := config.DB.Create(&commission).Error; err != nil {
				log.Printf("[ReviewSubmission] ERROR creating commission: %v", err)
			} else {
				log.Printf("[ReviewSubmission] SUCCESS created commission ID=%d for parent %d, amount %.2f", commission.ID, *user.ParentID, commission.Amount)
				// 将佣金加入邀请人余额
				var parent models.User
				if config.DB.First(&parent, *user.ParentID).Error == nil {
					parent.Balance += commission.Amount
					config.DB.Save(&parent)
					log.Printf("[ReviewSubmission] 佣金已计入邀请人余额: ParentID=%d, Amount=%.2f", *user.ParentID, commission.Amount)
				}
			}
		} else {
			log.Printf("[ReviewSubmission] User %d has NO parent, skipping commission", user.ID)
		}
	} else if req.Status == "rejected" {
		submission.RejectReason = req.RejectReason
	}

	// 清除关联对象以避免GORM尝试更新它们
	submission.Task = nil
	submission.User = nil

	if err := config.DB.Save(&submission).Error; err != nil {
		log.Printf("[ReviewSubmission] Error saving submission: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败"})
		return
	}

	log.Printf("[ReviewSubmission] Successfully updated submission ID=%s to status=%s", id, submission.Status)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "审核完成",
	})
}

// detectPlatformFromUrl 从URL检测平台类型
func detectPlatformFromUrl(link string) string {
	link = strings.ToLower(link)
	if strings.Contains(link, "xiaohongshu") || strings.Contains(link, "xhslink") || strings.Contains(link, "xhs") {
		return "xiaohongshu"
	}
	if strings.Contains(link, "douyin") || strings.Contains(link, "iesdouyin") {
		return "douyin"
	}
	if strings.Contains(link, "kuaishou") || strings.Contains(link, "kwai") || strings.Contains(link, "gifshow") {
		return "kuaishou"
	}
	if strings.Contains(link, "bilibili") || strings.Contains(link, "b23.tv") {
		return "bilibili"
	}
	if strings.Contains(link, "weixin") || strings.Contains(link, "wechat") || strings.Contains(link, "channels") {
		return "wechat_channels"
	}
	if strings.Contains(link, "weibo") {
		return "weibo"
	}
	return "unknown"
}

// checkLinkRealValidity 真实检查链接有效性 - 支持多平台
func checkLinkRealValidity(link string) (isValid bool, responseCode int, errorMsg string) {
	platform := detectPlatformFromUrl(link)

	client := &http.Client{
		Timeout: 15 * time.Second,
		// 跟随重定向
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return false, 0, "创建请求失败: " + err.Error()
	}

	// 设置浏览器User-Agent模拟正常访问
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	// 根据平台设置特定的Referer
	switch platform {
	case "xiaohongshu":
		req.Header.Set("Referer", "https://www.xiaohongshu.com/")
	case "douyin":
		req.Header.Set("Referer", "https://www.douyin.com/")
	case "kuaishou":
		req.Header.Set("Referer", "https://www.kuaishou.com/")
	case "bilibili":
		req.Header.Set("Referer", "https://www.bilibili.com/")
	case "wechat_channels":
		req.Header.Set("Referer", "https://channels.weixin.qq.com/")
	case "weibo":
		req.Header.Set("Referer", "https://weibo.com/")
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, 0, "请求失败: " + err.Error()
	}
	defer resp.Body.Close()

	responseCode = resp.StatusCode

	// 检查HTTP状态码 - 200/301/302/303/307/308 都视为有效
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, responseCode, ""
	}

	// 404或其他错误状态码视为无效
	return false, responseCode, fmt.Sprintf("HTTP状态码: %d", resp.StatusCode)
}

// CheckLinkValidity 检查链接有效性 (手动触发)
func CheckLinkValidity(c *gin.Context) {
	id := c.Param("id")

	var submission models.TaskSubmission
	if err := config.DB.First(&submission, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "提交不存在"})
		return
	}

	// 真实HTTP请求检查链接有效性
	isValid, responseCode, errorMsg := checkLinkRealValidity(submission.PublishLink)
	likesCount := submission.LikesCount // 点赞数暂时保持不变，小红书需要登录才能获取

	// 记录检查日志
	now := time.Now()
	log := models.LinkCheckLog{
		SubmissionID: submission.ID,
		CheckTime:    now,
		IsValid:      isValid,
		LikesCount:   likesCount,
		ResponseCode: responseCode,
		ErrorMessage: errorMsg,
	}
	config.DB.Create(&log)

	submission.LastCheckTime = &now
	submission.LinkValid = isValid
	submission.LikesCount = likesCount

	if !isValid {
		submission.Status = "invalid"
		// 拉黑用户
		var user models.User
		config.DB.First(&user, submission.UserID)
		user.Status = "blocked"
		config.DB.Save(&user)
	}

	config.DB.Save(&submission)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "链接检查完成",
		"data": gin.H{
			"is_valid":      isValid,
			"likes_count":   likesCount,
			"response_code": responseCode,
			"error_message": errorMsg,
		},
	})
}

// FetchLinkInfo 获取链接信息（点赞数、评论数等）
func FetchLinkInfo(c *gin.Context) {
	var req struct {
		Link string `json:"link" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请提供链接"})
		return
	}

	link := req.Link

	// HTTP客户端
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	httpReq, err := http.NewRequest("GET", link, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "请求创建失败",
			"data": gin.H{
				"success":   false,
				"error":     err.Error(),
				"link":      link,
				"likes":     0,
				"comments":  0,
				"collects":  0,
				"shares":    0,
				"title":     "",
				"author":    "",
				"is_valid":  false,
				"http_code": 0,
			},
		})
		return
	}

	// 设置浏览器UA模拟正常访问
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	httpReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	httpReq.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	httpReq.Header.Set("Referer", "https://www.xiaohongshu.com/")
	httpReq.Header.Set("Cookie", "xhsTrackerId=ceced5c5-5f16-4c30-9c9f-5d5f1d5f5d5f")

	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "请求失败",
			"data": gin.H{
				"success":   false,
				"error":     err.Error(),
				"link":      link,
				"likes":     0,
				"comments":  0,
				"collects":  0,
				"shares":    0,
				"title":     "",
				"author":    "",
				"is_valid":  false,
				"http_code": 0,
			},
		})
		return
	}
	defer resp.Body.Close()

	httpCode := resp.StatusCode
	isValid := httpCode >= 200 && httpCode < 400

	// 读取页面内容（限制大小）
	bodyBytes := make([]byte, 1024*100) // 读取前100KB
	n, _ := resp.Body.Read(bodyBytes)
	bodyStr := string(bodyBytes[:n])

	// 尝试提取信息（简单正则匹配）
	var likes, comments, collects, shares int
	var title, author string

	// 尝试从页面中提取数据
	// 小红书页面中通常有 "likeCount":xxx 格式的JSON数据
	// 提取点赞数
	if idx := findJSONValue(bodyStr, "likeCount"); idx != "" {
		fmt.Sscanf(idx, "%d", &likes)
	} else if idx := findJSONValue(bodyStr, "liked_count"); idx != "" {
		fmt.Sscanf(idx, "%d", &likes)
	}

	// 提取评论数
	if idx := findJSONValue(bodyStr, "commentCount"); idx != "" {
		fmt.Sscanf(idx, "%d", &comments)
	} else if idx := findJSONValue(bodyStr, "comments_count"); idx != "" {
		fmt.Sscanf(idx, "%d", &comments)
	}

	// 提取收藏数
	if idx := findJSONValue(bodyStr, "collectCount"); idx != "" {
		fmt.Sscanf(idx, "%d", &collects)
	} else if idx := findJSONValue(bodyStr, "collected_count"); idx != "" {
		fmt.Sscanf(idx, "%d", &collects)
	}

	// 提取分享数
	if idx := findJSONValue(bodyStr, "shareCount"); idx != "" {
		fmt.Sscanf(idx, "%d", &shares)
	}

	// 提取标题
	if idx := findJSONValue(bodyStr, "title"); idx != "" {
		title = idx
	}

	// 提取作者
	if idx := findJSONValue(bodyStr, "nickname"); idx != "" {
		author = idx
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"success":   isValid,
			"link":      link,
			"likes":     likes,
			"comments":  comments,
			"collects":  collects,
			"shares":    shares,
			"title":     title,
			"author":    author,
			"is_valid":  isValid,
			"http_code": httpCode,
		},
	})
}

// findJSONValue 从HTML/JSON中查找特定键的值
func findJSONValue(content, key string) string {
	// 查找 "key":value 或 "key":"value" 格式
	patterns := []string{
		`"` + key + `":`,
		`'` + key + `':`,
		key + `=`,
	}

	for _, pattern := range patterns {
		idx := -1
		for i := 0; i < len(content)-len(pattern); i++ {
			if content[i:i+len(pattern)] == pattern {
				idx = i + len(pattern)
				break
			}
		}

		if idx > 0 && idx < len(content) {
			// 跳过空格
			for idx < len(content) && (content[idx] == ' ' || content[idx] == '\t') {
				idx++
			}

			// 提取值
			if idx < len(content) {
				if content[idx] == '"' || content[idx] == '\'' {
					// 字符串值
					quote := content[idx]
					idx++
					end := idx
					for end < len(content) && content[end] != quote {
						end++
					}
					if end > idx {
						return content[idx:end]
					}
				} else {
					// 数字值
					end := idx
					for end < len(content) && (content[end] >= '0' && content[end] <= '9') {
						end++
					}
					if end > idx {
						return content[idx:end]
					}
				}
			}
		}
	}
	return ""
}

// ListMerchantUsers 用户列表 (商家)
func ListMerchantUsers(c *gin.Context) {
	var users []models.User
	config.DB.Where("role = ?", "user").Order("created_at DESC").Find(&users)

	var list []gin.H
	for _, u := range users {
		// 统计用户提交数
		var submissionCount int64
		config.DB.Model(&models.TaskSubmission{}).Where("user_id = ?", u.ID).Count(&submissionCount)

		list = append(list, gin.H{
			"id":               u.ID,
			"username":         u.Username,
			"nickname":         u.Nickname,
			"phone":            u.Phone,
			"status":           u.Status,
			"balance":          u.Balance,
			"submission_count": submissionCount,
			"created_at":       u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": list,
	})
}

// BlockUser 拉黑用户
func BlockUser(c *gin.Context) {
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

// ListWithdrawals 提现列表 (商家)
func ListMerchantWithdrawals(c *gin.Context) {
	status := c.Query("status")

	var withdrawals []models.Withdrawal
	query := config.DB.Model(&models.Withdrawal{}).Preload("User")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Order("created_at DESC").Find(&withdrawals)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": withdrawals,
	})
}

// ProcessWithdrawal 处理提现
func ProcessWithdrawal(c *gin.Context) {
	id := c.Param("id")

	var withdrawal models.Withdrawal
	if err := config.DB.First(&withdrawal, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "提现记录不存在"})
		return
	}

	var req struct {
		Status       string `json:"status" binding:"required"` // approved, paid, rejected
		RejectReason string `json:"reject_reason"`
		PayProof     string `json:"pay_proof"`
		Remark       string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 获取当前操作者ID
	merchantID, _ := c.Get("user_id")
	now := time.Now()

	withdrawal.Status = req.Status

	if req.Status == "approved" {
		withdrawal.ReviewedAt = &now
		if uid, ok := merchantID.(uint); ok {
			withdrawal.ReviewedBy = &uid
		}
		if req.Remark != "" {
			withdrawal.Remark = req.Remark
		}
	} else if req.Status == "paid" {
		withdrawal.PaidAt = &now
		withdrawal.PayProof = req.PayProof
		if req.Remark != "" {
			withdrawal.Remark = req.Remark
		}
	} else if req.Status == "rejected" {
		withdrawal.RejectReason = req.RejectReason
		withdrawal.ReviewedAt = &now
		if uid, ok := merchantID.(uint); ok {
			withdrawal.ReviewedBy = &uid
		}
		// 退回余额
		var user models.User
		config.DB.First(&user, withdrawal.UserID)
		user.Balance += withdrawal.Amount
		config.DB.Save(&user)
	}

	config.DB.Save(&withdrawal)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "处理成功",
	})
}

// ListPlatforms 平台列表
func ListPlatforms(c *gin.Context) {
	var platforms []models.Platform
	config.DB.Where("status = ?", "active").Find(&platforms)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": platforms,
	})
}

// ========== 订单管理 ==========

// ListMerchantOrders 商家订单列表
func ListMerchantOrders(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	offset := (page - 1) * pageSize

	query := config.DB.Model(&models.Order{}).
		Preload("User").
		Where("merchant_id = ?", merchantID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var orders []models.Order
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  orders,
			"total": total,
			"page":  page,
		},
	})
}

// GetOrderStats 订单统计
func GetOrderStats(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var totalOrders, completedOrders int64
	var totalAmount, totalCommission float64

	config.DB.Model(&models.Order{}).Where("merchant_id = ?", merchantID).Count(&totalOrders)
	config.DB.Model(&models.Order{}).Where("merchant_id = ? AND status = ?", merchantID, "completed").Count(&completedOrders)
	config.DB.Model(&models.Order{}).Where("merchant_id = ?", merchantID).Select("COALESCE(SUM(amount), 0)").Scan(&totalAmount)
	config.DB.Model(&models.Commission{}).
		Joins("JOIN users ON users.id = commissions.from_user_id").
		Where("users.merchant_id = ?", merchantID).
		Select("COALESCE(SUM(commissions.amount), 0)").Scan(&totalCommission)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total_orders":     totalOrders,
			"completed_orders": completedOrders,
			"total_amount":     totalAmount,
			"total_commission": totalCommission,
		},
	})
}

// ========== 分销管理 ==========

// GetDistributionStats 分销统计
func GetDistributionStats(c *gin.Context) {
	var totalInvitees, activeInvitees int64
	var totalCommission, paidCommission, level1Commission, level2Commission float64

	// 统计总邀请人数 (所有邀请记录)
	config.DB.Model(&models.Invitation{}).Count(&totalInvitees)

	// 统计活跃邀请人数（被邀请人有提交记录的）
	config.DB.Model(&models.User{}).
		Where("parent_id IS NOT NULL AND id IN (SELECT user_id FROM task_submissions)").
		Count(&activeInvitees)

	// 总佣金统计
	config.DB.Model(&models.Commission{}).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalCommission)

	// 已结算佣金统计
	config.DB.Model(&models.Commission{}).
		Where("status = ?", "settled").
		Select("COALESCE(SUM(amount), 0)").Scan(&paidCommission)

	// 一级分销佣金统计（直接邀请产生的佣金）
	// 一级佣金：被邀请人的parent_id = 佣金获得者的user_id
	config.DB.Model(&models.Commission{}).
		Joins("JOIN users ON users.id = commissions.from_user_id").
		Where("users.parent_id = commissions.user_id").
		Select("COALESCE(SUM(commissions.amount), 0)").Scan(&level1Commission)

	// 二级分销佣金统计（间接邀请产生的佣金）
	// 二级佣金：被邀请人的上级的parent_id = 佣金获得者的user_id
	config.DB.Model(&models.Commission{}).
		Joins("JOIN users u1 ON u1.id = commissions.from_user_id").
		Joins("JOIN users u2 ON u2.id = u1.parent_id").
		Where("u2.parent_id = commissions.user_id").
		Select("COALESCE(SUM(commissions.amount), 0)").Scan(&level2Commission)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"total_invitees":    totalInvitees,
			"active_invitees":   activeInvitees,
			"total_commission":  totalCommission,
			"paid_commission":   paidCommission,
			"level1_commission": level1Commission,
			"level2_commission": level2Commission,
		},
	})
}

// ListInvitations 邀请记录列表
func ListInvitations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	var total int64
	var invitations []models.Invitation

	config.DB.Model(&models.Invitation{}).Count(&total)

	config.DB.Model(&models.Invitation{}).
		Preload("Inviter").
		Preload("Invitee").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&invitations)

	// 为每个邀请记录计算佣金统计
	var invitationList []gin.H
	for _, inv := range invitations {
		// 计算该被邀请人产生的佣金金额（邀请人获得的）
		var commissionFromInvitee float64
		config.DB.Model(&models.Commission{}).
			Where("user_id = ? AND from_user_id = ?", inv.InviterID, inv.InviteeID).
			Select("COALESCE(SUM(amount), 0)").Scan(&commissionFromInvitee)

		// 计算被邀请人完成的任务数
		var submissionCount int64
		config.DB.Model(&models.TaskSubmission{}).
			Where("user_id = ? AND status = ?", inv.InviteeID, "approved").
			Count(&submissionCount)

		// 计算被邀请人发展的下级数量
		var subInviteeCount int64
		config.DB.Model(&models.Invitation{}).
			Where("inviter_id = ?", inv.InviteeID).
			Count(&subInviteeCount)

		inviterName := ""
		if inv.Inviter != nil {
			inviterName = inv.Inviter.Nickname
			if inviterName == "" {
				inviterName = inv.Inviter.Username
			}
		}

		inviteeName := ""
		if inv.Invitee != nil {
			inviteeName = inv.Invitee.Nickname
			if inviteeName == "" {
				inviteeName = inv.Invitee.Username
			}
		}

		invitationList = append(invitationList, gin.H{
			"id":                  inv.ID,
			"inviter_id":          inv.InviterID,
			"invitee_id":          inv.InviteeID,
			"inviter_name":        inviterName,
			"invitee_name":        inviteeName,
			"invite_code":         inv.InviteCode,
			"created_at":          inv.CreatedAt,
			"commission_earned":   commissionFromInvitee,
			"invitee_submissions": submissionCount,
			"invitee_sub_count":   subInviteeCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  invitationList,
			"total": total,
			"page":  page,
		},
	})
}

// ========== 系统设置 ==========

// GetMerchantConfig 获取商家配置
func GetMerchantConfig(c *gin.Context) {
	// 获取系统默认配置
	var configs []models.SystemConfig
	config.DB.Find(&configs)

	configMap := make(map[string]string)
	for _, cfg := range configs {
		configMap[cfg.ConfigKey] = cfg.ConfigValue
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": configMap,
	})
}

// UpdateMerchantConfig 更新商家配置
func UpdateMerchantConfig(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 只允许修改特定配置项
	allowedKeys := []string{"commission_rate", "min_withdraw", "withdraw_fee", "link_check_interval", "auto_review_days"}
	allowed := false
	for _, k := range allowedKeys {
		if k == req.Key {
			allowed = true
			break
		}
	}

	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "不允许修改此配置项"})
		return
	}

	var cfg models.SystemConfig
	result := config.DB.Where("config_key = ?", req.Key).First(&cfg)
	if result.Error != nil {
		cfg = models.SystemConfig{
			ConfigKey:   req.Key,
			ConfigValue: req.Value,
		}
		config.DB.Create(&cfg)
	} else {
		cfg.ConfigValue = req.Value
		config.DB.Save(&cfg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "配置更新成功",
	})
}
