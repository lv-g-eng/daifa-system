package handlers

import (
	"distribution-system/config"
	"distribution-system/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// fmtTime 安全格式化时间指针
func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// applyDateRange 按 query 的 start/end（YYYY-MM-DD）过滤 created_at
func applyDateRange(q *gorm.DB, c *gin.Context) *gorm.DB {
	if start := c.Query("start"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if end := c.Query("end"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			q = q.Where("created_at < ?", t.AddDate(0, 0, 1)) // 含当天
		}
	}
	return q
}

// writeExcel 通用：表头 + 数据行写入 xlsx，作为附件下载
func writeExcel(c *gin.Context, filename string, headers []string, rows [][]interface{}) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Sheet1"

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFE8EC"}, Pattern: 1},
	})
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
	}
	for r, row := range rows {
		for ci, val := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, r+2)
			f.SetCellValue(sheet, cell, val)
		}
	}
	if len(headers) > 0 {
		lastCol, _ := excelize.ColumnNumberToName(len(headers))
		f.SetColWidth(sheet, "A", lastCol, 16)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "导出失败"})
	}
}

func dateTag() string { return time.Now().Format("20060102") }

// ===================== 商家导出 =====================

// ExportMerchantTasks 导出当前商家的任务报表
func ExportMerchantTasks(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var tasks []models.Task
	q := config.DB.Model(&models.Task{}).Where("merchant_id = ?", merchantID).Preload("Platform")
	q = applyDateRange(q, c)
	q.Order("created_at DESC").Find(&tasks)

	headers := []string{"任务ID", "项目名称", "平台", "单价", "团长价", "奖励", "有效天数", "总数", "已完成", "状态", "自动审核", "创建时间"}
	rows := make([][]interface{}, 0, len(tasks))
	for _, t := range tasks {
		platform := ""
		if t.Platform != nil {
			platform = t.Platform.Name
		}
		autoReview := "否"
		if t.AutoReview {
			autoReview = "是"
		}
		rows = append(rows, []interface{}{
			t.ID, t.ProjectName, platform, t.UnitPrice, t.TeamLeaderPrice, t.BonusPrice,
			t.ValidityDays, t.TotalCount, t.CompletedCount, statusZH(t.Status), autoReview,
			t.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	writeExcel(c, "tasks_"+dateTag()+".xlsx", headers, rows)
}

// ExportMerchantWithdrawals 导出提现报表（管理端复用）
func ExportMerchantWithdrawals(c *gin.Context) {
	status := c.Query("status")
	q := config.DB.Model(&models.Withdrawal{}).Preload("User")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q = applyDateRange(q, c)

	var ws []models.Withdrawal
	q.Order("created_at DESC").Find(&ws)

	headers := []string{"提现ID", "用户", "金额", "收款方式", "收款账号", "状态", "拒绝原因", "申请时间", "审核时间"}
	rows := make([][]interface{}, 0, len(ws))
	for _, w := range ws {
		username := ""
		if w.User != nil {
			username = w.User.Username
		}
		rows = append(rows, []interface{}{
			w.ID, username, w.Amount, w.PaymentMethod, w.PaymentAccount,
			statusZH(w.Status), w.RejectReason, w.CreatedAt.Format("2006-01-02 15:04"), fmtTime(w.ReviewedAt),
		})
	}
	writeExcel(c, "withdrawals_"+dateTag()+".xlsx", headers, rows)
}

// ExportMerchantCommissions 导出分销佣金报表
func ExportMerchantCommissions(c *gin.Context) {
	q := config.DB.Model(&models.Commission{}).Preload("User").Preload("FromUser")
	q = applyDateRange(q, c)

	var cs []models.Commission
	q.Order("created_at DESC").Find(&cs)

	headers := []string{"佣金ID", "收益用户", "来源用户", "金额", "比例(%)", "状态", "时间"}
	rows := make([][]interface{}, 0, len(cs))
	for _, cm := range cs {
		to, from := "", ""
		if cm.User != nil {
			to = cm.User.Username
		}
		if cm.FromUser != nil {
			from = cm.FromUser.Username
		}
		rows = append(rows, []interface{}{
			cm.ID, to, from, cm.Amount, cm.Rate, statusZH(cm.Status),
			cm.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	writeExcel(c, "commissions_"+dateTag()+".xlsx", headers, rows)
}

// ===================== 管理端导出 =====================

// ExportAdminUsers 导出全部普通用户
func ExportAdminUsers(c *gin.Context) {
	q := config.DB.Model(&models.User{}).Where("role = ?", "user")
	if kw := c.Query("keyword"); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("username LIKE ? OR nickname LIKE ? OR phone LIKE ?", like, like, like)
	}
	q = applyDateRange(q, c)

	var users []models.User
	q.Order("created_at DESC").Find(&users)

	headers := []string{"用户ID", "用户名", "昵称", "手机", "余额", "状态", "注册时间"}
	rows := make([][]interface{}, 0, len(users))
	for _, u := range users {
		rows = append(rows, []interface{}{
			u.ID, u.Username, u.Nickname, u.Phone, u.Balance, statusZH(u.Status),
			u.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	writeExcel(c, "users_"+dateTag()+".xlsx", headers, rows)
}

// ExportAdminMerchants 导出全部商家
func ExportAdminMerchants(c *gin.Context) {
	q := config.DB.Model(&models.User{}).Where("role = ?", "merchant")
	if kw := c.Query("keyword"); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("username LIKE ? OR nickname LIKE ? OR phone LIKE ?", like, like, like)
	}
	q = applyDateRange(q, c)

	var ms []models.User
	q.Order("created_at DESC").Find(&ms)

	headers := []string{"商家ID", "用户名", "昵称", "手机", "状态", "到期时间", "创建时间"}
	rows := make([][]interface{}, 0, len(ms))
	for _, m := range ms {
		rows = append(rows, []interface{}{
			m.ID, m.Username, m.Nickname, m.Phone, statusZH(m.Status), fmtTime(m.ExpireTime),
			m.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	writeExcel(c, "merchants_"+dateTag()+".xlsx", headers, rows)
}

// statusZH 状态英文转中文（导出可读性）
func statusZH(s string) string {
	switch s {
	case "active":
		return "进行中"
	case "pending":
		return "待处理"
	case "approved":
		return "已通过"
	case "rejected":
		return "已拒绝"
	case "paid":
		return "已打款"
	case "completed":
		return "已完成"
	case "invalid":
		return "已失效"
	case "expired":
		return "已过期"
	case "blocked":
		return "已封禁"
	default:
		return s
	}
}
