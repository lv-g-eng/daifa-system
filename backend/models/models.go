package models

import (
	"time"
)

// User 用户模型
type User struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Username        string     `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password        string     `gorm:"size:255;not null" json:"-"`
	Nickname        string     `gorm:"size:100" json:"nickname"`
	Phone           string     `gorm:"size:20" json:"phone"`
	Avatar          string     `gorm:"size:500" json:"avatar"`
	Role            string     `gorm:"size:20;not null;default:user" json:"role"`     // admin, merchant, user
	Status          string     `gorm:"size:20;not null;default:active" json:"status"` // active, blocked, expired
	Balance         float64    `gorm:"type:decimal(10,2);default:0" json:"balance"`
	PaymentQRCode   string     `gorm:"column:payment_qrcode;size:500" json:"payment_qrcode"`
	PaymentVerified bool       `gorm:"default:false" json:"payment_verified"`
	ExpireTime      *time.Time `gorm:"column:expire_time" json:"expire_time"`
	ParentID        *uint      `gorm:"column:parent_id" json:"parent_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Platform 平台模型
type Platform struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"size:50;not null" json:"name"`
	Code       string    `gorm:"uniqueIndex;size:20;not null" json:"code"`
	Icon       string    `gorm:"size:500" json:"icon"`
	PublishURL string    `gorm:"column:publish_url;size:500" json:"publish_url"`
	Status     string    `gorm:"size:20;default:active" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// Material 素材模型
type Material struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	MerchantID    uint      `gorm:"not null;index" json:"merchant_id"`
	MaterialNo    string    `gorm:"size:50;not null;index" json:"material_no"`
	PlatformID    *uint     `gorm:"index" json:"platform_id"`
	Category      string    `gorm:"size:100" json:"category"`
	SequenceNo    int       `json:"sequence_no"`
	Title         string    `gorm:"size:200;not null" json:"title"`
	Content       string    `gorm:"type:text" json:"content"`
	Topics        string    `gorm:"size:500" json:"topics"`
	Images        string    `gorm:"type:text" json:"images"`
	Videos        string    `gorm:"type:text" json:"videos"`
	Audios        string    `gorm:"type:text" json:"audios"`
	DownloadCount int       `gorm:"default:0" json:"download_count"`
	Status        string    `gorm:"size:20;default:active" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// 关联
	Platform *Platform `gorm:"foreignKey:PlatformID" json:"platform,omitempty"`
}

// PublishCode 发布码模型
type PublishCode struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	MerchantID  uint       `gorm:"not null;index" json:"merchant_id"`
	MaterialID  *uint      `json:"material_id"`
	PlatformID  uint       `gorm:"not null" json:"platform_id"`
	ProjectName string     `gorm:"size:100" json:"project_name"`
	Code        string     `gorm:"uniqueIndex;size:100;not null" json:"code"`
	QRCodeURL   string     `gorm:"column:qrcode_url;size:500" json:"qrcode_url"`
	MaxScans    int        `gorm:"default:1" json:"max_scans"`
	ScanCount   int        `gorm:"default:0" json:"scan_count"`
	ExpireTime  *time.Time `json:"expire_time"`
	Status      string     `gorm:"size:20;default:active" json:"status"`
	CreatedAt   time.Time  `json:"created_at"`

	// 关联
	Material *Material `gorm:"foreignKey:MaterialID" json:"material,omitempty"`
	Platform *Platform `gorm:"foreignKey:PlatformID" json:"platform,omitempty"`
}

// Task 任务模型
type Task struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	MerchantID      uint       `gorm:"not null;index" json:"merchant_id"`
	PlatformID      uint       `gorm:"not null;index" json:"platform_id"`
	ProjectName     string     `gorm:"size:200;not null" json:"project_name"`
	TeamGroup       string     `gorm:"size:100" json:"team_group"`
	Description     string     `gorm:"type:text" json:"description"`
	ExampleImages   string     `gorm:"type:text" json:"example_images"`
	UnitPrice       float64    `gorm:"type:decimal(10,2);not null" json:"unit_price"`
	TeamLeaderPrice float64    `gorm:"type:decimal(10,2);default:0" json:"team_leader_price"`
	BonusPrice      float64    `gorm:"type:decimal(10,2);default:0" json:"bonus_price"`
	Deadline        *time.Time `json:"deadline"`
	ValidityDays    int        `gorm:"default:30" json:"validity_days"`
	TotalCount      int        `gorm:"default:0" json:"total_count"`
	CompletedCount  int        `gorm:"default:0" json:"completed_count"`
	AutoReview      bool       `gorm:"default:false" json:"auto_review"`
	SampleRate      int        `gorm:"default:0" json:"sample_rate"`
	AutoReviewDays  int        `gorm:"default:7" json:"auto_review_days"` // 自动审核天数，链接发布后N天自动审核通过（1-30天）
	Status          string     `gorm:"size:20;default:active;index" json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// 关联
	Platform *Platform `gorm:"foreignKey:PlatformID" json:"platform,omitempty"`
	Merchant *User     `gorm:"foreignKey:MerchantID" json:"merchant,omitempty"`
}

// TaskSubmission 任务提交模型
type TaskSubmission struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	TaskID           uint       `gorm:"not null;index" json:"task_id"`
	UserID           uint       `gorm:"not null;index" json:"user_id"`
	MaterialNo       string     `gorm:"size:50" json:"material_no"`
	PublishLink      string     `gorm:"size:1000;not null" json:"publish_link"`
	ScreenshotURL    string     `gorm:"size:500" json:"screenshot_url"`
	LikesCount       int        `gorm:"default:0" json:"likes_count"`
	LastCheckTime    *time.Time `json:"last_check_time"`
	LinkValid        bool       `gorm:"default:true;index" json:"link_valid"`
	ValidityExpire   *time.Time `gorm:"index" json:"validity_expire"`
	Status           string     `gorm:"size:20;default:pending;index" json:"status"` // pending, approved, rejected, expired, invalid
	RejectReason     string     `gorm:"size:500" json:"reject_reason"`
	UnitPrice        float64    `gorm:"type:decimal(10,2)" json:"unit_price"`
	TeamLeaderAmount float64    `gorm:"type:decimal(10,2);default:0" json:"team_leader_amount"`
	BonusAmount      float64    `gorm:"type:decimal(10,2);default:0" json:"bonus_amount"`
	TotalAmount      float64    `gorm:"type:decimal(10,2);default:0" json:"total_amount"`
	Settled          bool       `gorm:"default:false" json:"settled"`
	SettledAt        *time.Time `json:"settled_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// 关联
	Task *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// LinkCheckLog 链接检查日志
type LinkCheckLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	SubmissionID uint      `gorm:"not null;index" json:"submission_id"`
	CheckTime    time.Time `gorm:"not null;index" json:"check_time"`
	IsValid      bool      `gorm:"not null" json:"is_valid"`
	LikesCount   int       `gorm:"default:0" json:"likes_count"`
	ResponseCode int       `json:"response_code"`
	ErrorMessage string    `gorm:"size:500" json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
}

// Order 订单模型
type Order struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	OrderNo      string    `gorm:"uniqueIndex;size:50;not null" json:"order_no"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	MerchantID   uint      `gorm:"not null;index" json:"merchant_id"`
	SubmissionID *uint     `json:"submission_id"`
	OrderType    string    `gorm:"size:20;not null" json:"order_type"` // task_reward, commission, withdrawal
	Amount       float64   `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status       string    `gorm:"size:20;default:pending" json:"status"`
	Remark       string    `gorm:"size:500" json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Withdrawal 提现记录模型
type Withdrawal struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	Amount         float64    `gorm:"type:decimal(10,2);not null" json:"amount"`
	PaymentMethod  string     `gorm:"size:20;default:alipay" json:"payment_method"` // alipay, wechat
	PaymentAccount string     `gorm:"size:100" json:"payment_account"`
	PaymentQRCode  string     `gorm:"column:payment_qrcode;size:500" json:"payment_qrcode"`
	Status         string     `gorm:"size:20;default:pending;index" json:"status"` // pending, approved, paid, rejected
	RejectReason   string     `gorm:"size:500" json:"reject_reason"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	ReviewedBy     *uint      `json:"reviewed_by"`
	PaidAt         *time.Time `json:"paid_at"`
	PayProof       string     `gorm:"size:500" json:"pay_proof"`
	Remark         string     `gorm:"size:500" json:"remark"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Commission 分销佣金模型
type Commission struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"not null;index" json:"user_id"`
	FromUserID   uint       `gorm:"not null;index" json:"from_user_id"`
	SubmissionID *uint      `json:"submission_id"`
	Amount       float64    `gorm:"type:decimal(10,2);not null" json:"amount"`
	Rate         float64    `gorm:"column:commission_rate;type:decimal(5,2)" json:"commission_rate"`
	Status       string     `gorm:"size:20;default:pending" json:"status"`
	SettledAt    *time.Time `json:"settled_at"`
	CreatedAt    time.Time  `json:"created_at"`

	// 关联
	User     *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	FromUser *User `gorm:"foreignKey:FromUserID" json:"from_user,omitempty"`
}

// Invitation 邀请关系模型
type Invitation struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	InviterID  uint      `gorm:"not null;index" json:"inviter_id"`
	InviteeID  uint      `gorm:"uniqueIndex;not null" json:"invitee_id"`
	InviteCode string    `gorm:"size:50" json:"invite_code"`
	CreatedAt  time.Time `json:"created_at"`

	// 关联
	Inviter *User `gorm:"foreignKey:InviterID" json:"inviter,omitempty"`
	Invitee *User `gorm:"foreignKey:InviteeID" json:"invitee,omitempty"`
}

// SystemConfig 系统配置模型
type SystemConfig struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MerchantID  *uint     `json:"merchant_id"`
	ConfigKey   string    `gorm:"size:100;not null" json:"config_key"`
	ConfigValue string    `gorm:"type:text" json:"config_value"`
	Description string    `gorm:"size:200" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string           { return "users" }
func (Platform) TableName() string       { return "platforms" }
func (Material) TableName() string       { return "materials" }
func (PublishCode) TableName() string    { return "publish_codes" }
func (Task) TableName() string           { return "tasks" }
func (TaskSubmission) TableName() string { return "task_submissions" }
func (LinkCheckLog) TableName() string   { return "link_check_logs" }
func (Order) TableName() string          { return "orders" }
func (Withdrawal) TableName() string     { return "withdrawals" }
func (Commission) TableName() string     { return "commissions" }
func (Invitation) TableName() string     { return "invitations" }
func (SystemConfig) TableName() string   { return "system_config" }
