package handlers

import (
	"distribution-system/config"
	"distribution-system/models"
	"fmt"
	"log"
	"time"
)

// StartScheduledTasks 启动定时任务
func StartScheduledTasks() {
	// 从数据库读取链接检查间隔配置
	go func() {
		for {
			// 每次执行前读取最新的配置
			// 兼容两个历史配置键：优先 link_check_interval，回退 auto_check_interval（init.sql 种子用的是后者）
			var cfg models.SystemConfig
			if config.DB.Where("config_key = ?", "link_check_interval").First(&cfg).Error != nil || cfg.ConfigValue == "" {
				config.DB.Where("config_key = ?", "auto_check_interval").First(&cfg)
			}
			interval := 1 // 默认1小时
			if cfg.ConfigValue != "" {
				fmt.Sscanf(cfg.ConfigValue, "%d", &interval)
			}
			if interval < 1 {
				interval = 1
			}
			if interval > 168 {
				interval = 168 // 最大7天
			}

			// 执行定时任务
			runScheduledTasks()

			// 等待配置的时间间隔
			time.Sleep(time.Duration(interval) * time.Hour)
		}
	}()
	log.Println("定时任务已启动：根据配置的间隔检查链接有效性和自动审核")
}

// runScheduledTasks 执行定时任务
func runScheduledTasks() {
	log.Println("开始执行定时任务...")

	// 1. 自动审核到期的提交
	autoReviewSubmissions()

	// 2. 检查已通过提交的链接有效性，30天内失效则拉黑用户
	checkApprovedLinksValidity()

	log.Println("定时任务执行完成")
}

// autoReviewSubmissions 自动审核到期的提交
func autoReviewSubmissions() {
	// 查找启用了自动审核的任务下的待审核提交
	var submissions []models.TaskSubmission

	config.DB.Model(&models.TaskSubmission{}).
		Preload("Task").
		Joins("JOIN tasks ON tasks.id = task_submissions.task_id").
		Where("task_submissions.status = ?", "pending").
		Where("tasks.auto_review = ?", true).
		Find(&submissions)

	now := time.Now()
	autoReviewCount := 0

	for _, sub := range submissions {
		if sub.Task == nil {
			continue
		}

		// 检查是否达到自动审核天数
		autoReviewDays := sub.Task.AutoReviewDays
		if autoReviewDays <= 0 {
			autoReviewDays = 7 // 默认7天
		}

		autoReviewTime := sub.CreatedAt.AddDate(0, 0, autoReviewDays)

		if now.After(autoReviewTime) {
			// 抽样检查
			shouldCheck := false
			if sub.Task.SampleRate > 0 {
				// 根据抽样率决定是否需要人工审核
				if autoReviewCount%100 < sub.Task.SampleRate {
					shouldCheck = true
				}
			}

			if shouldCheck {
				// 需要人工审核，跳过自动处理
				continue
			}

			// 自动审核通过
			sub.Status = "approved"
			validityExpire := now.AddDate(0, 0, sub.Task.ValidityDays)
			sub.ValidityExpire = &validityExpire
			sub.UnitPrice = sub.Task.UnitPrice
			sub.TeamLeaderAmount = sub.Task.TeamLeaderPrice
			sub.BonusAmount = sub.Task.BonusPrice
			sub.TotalAmount = sub.Task.UnitPrice + sub.Task.TeamLeaderPrice + sub.Task.BonusPrice

			// 更新用户余额
			var user models.User
			if err := config.DB.First(&user, sub.UserID).Error; err == nil {
				user.Balance += sub.TotalAmount
				config.DB.Save(&user)
			}

			// 更新任务完成数
			var task models.Task
			if err := config.DB.First(&task, sub.TaskID).Error; err == nil {
				task.CompletedCount++
				config.DB.Save(&task)
			}

			// 处理分销佣金
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
					SubmissionID: &sub.ID,
					Amount:       sub.TotalAmount * rate / 100,
					Rate:         rate,
					Status:       "pending",
				}
				config.DB.Create(&commission)
			}

			config.DB.Save(&sub)
			autoReviewCount++
			log.Printf("自动审核通过: 提交ID=%d, 用户ID=%d", sub.ID, sub.UserID)
		}
	}

	if autoReviewCount > 0 {
		log.Printf("本次自动审核通过 %d 个提交", autoReviewCount)
	}
}

// checkApprovedLinksValidity 检查已通过提交的链接有效性
func checkApprovedLinksValidity() {
	// 查找30天有效期内的已通过提交
	now := time.Now()

	var submissions []models.TaskSubmission
	config.DB.Model(&models.TaskSubmission{}).
		Where("status = ?", "approved").
		Where("validity_expire IS NOT NULL AND validity_expire > ?", now).
		Where("link_valid = ?", true).
		Find(&submissions)

	invalidCount := 0
	blockedUsers := make(map[uint]bool)

	for _, sub := range submissions {
		// 检查链接有效性
		isValid, responseCode, errorMsg := checkLinkRealValidity(sub.PublishLink)

		// 记录检查日志
		checkTime := time.Now()
		log := models.LinkCheckLog{
			SubmissionID: sub.ID,
			CheckTime:    checkTime,
			IsValid:      isValid,
			LikesCount:   sub.LikesCount,
			ResponseCode: responseCode,
			ErrorMessage: errorMsg,
		}
		config.DB.Create(&log)

		sub.LastCheckTime = &checkTime
		sub.LinkValid = isValid

		if !isValid {
			// 链接失效
			sub.Status = "invalid"
			invalidCount++

			// 30天内链接失效，拉黑用户
			if sub.ValidityExpire != nil && sub.ValidityExpire.After(now) {
				if !blockedUsers[sub.UserID] {
					var user models.User
					if err := config.DB.First(&user, sub.UserID).Error; err == nil {
						user.Status = "blocked"
						config.DB.Save(&user)
						blockedUsers[sub.UserID] = true
						log2 := fmt.Sprintf("用户因30天内链接失效被自动拉黑: 用户ID=%d, 提交ID=%d", sub.UserID, sub.ID)
						fmt.Println(log2)
					}
				}
			}
		}

		config.DB.Save(&sub)
	}

	if invalidCount > 0 {
		log.Printf("本次检查发现 %d 个无效链接，拉黑 %d 个用户", invalidCount, len(blockedUsers))
	}
}

