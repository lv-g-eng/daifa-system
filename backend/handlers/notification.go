package handlers

import (
	"distribution-system/config"
	"distribution-system/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateNotification 创建一条用户通知（内部调用，失败仅记录不阻断主流程）
func CreateNotification(userID uint, title, content, ntype string) {
	if userID == 0 {
		return
	}
	if ntype == "" {
		ntype = "system"
	}
	n := models.Notification{
		UserID:  userID,
		Title:   title,
		Content: content,
		Type:    ntype,
	}
	config.DB.Create(&n)
}

// ListNotifications 通知列表（当前用户，倒序，支持分页 + 未读数）
func ListNotifications(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	onlyUnread := c.Query("unread") == "1"

	query := config.DB.Model(&models.Notification{}).Where("user_id = ?", uid)
	if onlyUnread {
		query = query.Where("is_read = ?", false)
	}

	var total int64
	query.Count(&total)

	var list []models.Notification
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list)

	var unread int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", uid, false).
		Count(&unread)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":         list,
			"total":        total,
			"unread_count": unread,
			"page":         page,
			"page_size":    pageSize,
		},
	})
}

// UnreadNotificationCount 未读通知数（用于角标）
func UnreadNotificationCount(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	var unread int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", uid, false).
		Count(&unread)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"unread_count": unread},
	})
}

// MarkNotificationRead 标记单条通知为已读（id 通过 query 传，避免与静态路由冲突）
func MarkNotificationRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 id 参数"})
		return
	}

	config.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, uid).
		Update("is_read", true)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "已读"})
}

// MarkAllNotificationsRead 全部标记已读
func MarkAllNotificationsRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", uid, false).
		Update("is_read", true)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "全部已读"})
}

// BroadcastNotification 管理员群发站内通知（目标：user / merchant / all）
func BroadcastNotification(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Target  string `json:"target"` // user, merchant, all（默认 user）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	query := config.DB.Model(&models.User{})
	switch req.Target {
	case "merchant":
		query = query.Where("role = ?", "merchant")
	case "all":
		// 不加角色过滤，发给全部账号
	default:
		query = query.Where("role = ?", "user")
	}

	var users []models.User
	query.Find(&users)
	for _, u := range users {
		CreateNotification(u.ID, req.Title, req.Content, "system")
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "已群发",
		"data":    gin.H{"count": len(users)},
	})
}
