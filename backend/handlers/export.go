package handlers

import (
	"distribution-system/config"
	"distribution-system/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// fmtTime 安全格式化时间指针
func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// writeExcel 通用：表头 + 数据行写入 xlsx，作为附件下载
func writeExcel(c *gin.Context, filename string, headers []string, rows [][]interface{}) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Sheet1"

	// 表头（加粗）
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFE8EC"}, Pattern: 1},
	})
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
	}
	// 数据行
	for r, row := range rows {
		for ci, val := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, r+2)
			f.SetCellValue(sheet, cell, val)
		}
	}
	// 列宽自适应（粗略）
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

// ExportMerchantTasks 导出当前商家的任务报表
func ExportMerchantTasks(c *gin.Context) {
	userID, _ := c.Get("userID")
	merchantID := userID.(uint)

	var tasks []models.Task
	config.DB.Where("merchant_id = ?", merchantID).Preload("Platform").
		Order("created_at DESC").Find(&tasks)

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

// ExportMerchantWithdrawals 导出提现报表
func ExportMerchantWithdrawals(c *gin.Context) {
	status := c.Query("status")
	var ws []models.Withdrawal
	q := config.DB.Model(&models.Withdrawal{}).Preload("User")
	if status != "" {
		q = q.Where("status = ?", status)
	}
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
	var cs []models.Commission
	config.DB.Model(&models.Commission{}).Preload("User").Preload("FromUser").
		Order("created_at DESC").Find(&cs)

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
