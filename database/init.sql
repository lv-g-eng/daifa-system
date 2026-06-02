-- =====================================================
-- H5 代发系统数据库初始化脚本
-- 数据库: distribution_system
-- 用户: root / 密码: 123456
-- =====================================================

-- 创建数据库
CREATE DATABASE IF NOT EXISTS distribution_system DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE distribution_system;

-- =====================================================
-- 1. 用户表 (总管理员、商家、普通用户)
-- =====================================================
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL UNIQUE COMMENT '用户名',
    password VARCHAR(255) NOT NULL COMMENT '密码(明码存储)',
    nickname VARCHAR(100) COMMENT '昵称',
    phone VARCHAR(20) COMMENT '手机号',
    avatar VARCHAR(500) COMMENT '头像URL',
    role ENUM('admin', 'merchant', 'user') NOT NULL DEFAULT 'user' COMMENT '角色',
    status ENUM('active', 'blocked', 'expired') NOT NULL DEFAULT 'active' COMMENT '状态',
    balance DECIMAL(10,2) DEFAULT 0.00 COMMENT '账户余额',
    payment_qrcode VARCHAR(500) COMMENT '收款码图片URL',
    payment_verified BOOLEAN DEFAULT FALSE COMMENT '收款码是否已验证',
    expire_time DATETIME COMMENT '账号过期时间(商家用)',
    parent_id BIGINT COMMENT '上级用户ID(分销)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_role (role),
    INDEX idx_status (status),
    INDEX idx_parent_id (parent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- =====================================================
-- 2. 平台配置表 (支持多平台扩展,当前主要小红书)
-- =====================================================
CREATE TABLE IF NOT EXISTS platforms (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL COMMENT '平台名称',
    code VARCHAR(20) NOT NULL UNIQUE COMMENT '平台代码(xiaohongshu等)',
    icon VARCHAR(500) COMMENT '平台图标',
    publish_url VARCHAR(500) COMMENT '发布跳转URL',
    status ENUM('active', 'inactive') DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='平台配置表';

-- =====================================================
-- 3. 素材内容表
-- =====================================================
CREATE TABLE IF NOT EXISTS materials (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    merchant_id BIGINT NOT NULL COMMENT '所属商家ID',
    material_no VARCHAR(50) NOT NULL COMMENT '素材编号',
    platform_id BIGINT COMMENT '所属平台ID',
    category VARCHAR(100) COMMENT '所属赛道',
    sequence_no INT COMMENT '总序号',
    title VARCHAR(200) NOT NULL COMMENT '标题',
    content TEXT COMMENT '正文内容',
    topics VARCHAR(500) COMMENT '话题(逗号分隔)',
    images TEXT COMMENT '图片URLs(JSON数组)',
    videos TEXT COMMENT '视频URLs(JSON数组)',
    audios TEXT COMMENT '原音URLs(JSON数组)',
    download_count INT DEFAULT 0 COMMENT '下载次数',
    status ENUM('active', 'inactive') DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_merchant (merchant_id),
    INDEX idx_platform (platform_id),
    INDEX idx_category (category),
    INDEX idx_material_no (material_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='素材内容表';

-- =====================================================
-- 4. 发布码表
-- =====================================================
CREATE TABLE IF NOT EXISTS publish_codes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    merchant_id BIGINT NOT NULL COMMENT '所属商家ID',
    material_id BIGINT COMMENT '关联素材ID',
    platform_id BIGINT NOT NULL COMMENT '目标平台ID',
    code VARCHAR(100) NOT NULL UNIQUE COMMENT '发布码',
    qrcode_url VARCHAR(500) COMMENT '二维码图片URL',
    max_scans INT DEFAULT 1 COMMENT '最大扫码次数',
    scan_count INT DEFAULT 0 COMMENT '已扫码次数',
    expire_time DATETIME COMMENT '过期时间',
    status ENUM('active', 'used', 'expired') DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_merchant (merchant_id),
    INDEX idx_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布码表';

-- =====================================================
-- 5. 任务表
-- =====================================================
CREATE TABLE IF NOT EXISTS tasks (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    merchant_id BIGINT NOT NULL COMMENT '发布商家ID',
    platform_id BIGINT NOT NULL COMMENT '所属平台ID',
    project_name VARCHAR(200) NOT NULL COMMENT '项目名称',
    team_group VARCHAR(100) COMMENT '团队小组',
    description TEXT COMMENT '任务说明',
    example_images TEXT COMMENT '示例图(JSON数组)',
    unit_price DECIMAL(10,2) NOT NULL COMMENT '单价',
    team_leader_price DECIMAL(10,2) DEFAULT 0 COMMENT '团长金额',
    bonus_price DECIMAL(10,2) DEFAULT 0 COMMENT '引流奖励',
    deadline DATETIME COMMENT '执行时限',
    validity_days INT DEFAULT 30 COMMENT '链接有效期(天)',
    total_count INT DEFAULT 0 COMMENT '任务总数',
    completed_count INT DEFAULT 0 COMMENT '已完成数',
    auto_review BOOLEAN DEFAULT FALSE COMMENT '是否自动审核',
    sample_rate INT DEFAULT 0 COMMENT '抽查比例(%)',
    auto_review_days INT DEFAULT 7 COMMENT '自动审核天数(链接发布后N天自动通过,1-30天)',
    status ENUM('active', 'paused', 'completed', 'closed') DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_merchant (merchant_id),
    INDEX idx_platform (platform_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务表';

-- =====================================================
-- 6. 任务提交表 (链接回传)
-- =====================================================
CREATE TABLE IF NOT EXISTS task_submissions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_id BIGINT NOT NULL COMMENT '任务ID',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    material_no VARCHAR(50) COMMENT '素材编号',
    publish_link VARCHAR(1000) NOT NULL COMMENT '发布链接',
    screenshot_url VARCHAR(500) COMMENT '截亏过程截图',
    likes_count INT DEFAULT 0 COMMENT '点赞数',
    last_check_time DATETIME COMMENT '最后检查时间',
    link_valid BOOLEAN DEFAULT TRUE COMMENT '链接是否有效',
    validity_expire DATETIME COMMENT '有效期截止时间(30天)',
    status ENUM('pending', 'approved', 'rejected', 'expired', 'invalid') DEFAULT 'pending' COMMENT '审核状态',
    reject_reason VARCHAR(500) COMMENT '驳回原因',
    unit_price DECIMAL(10,2) COMMENT '结算单价',
    team_leader_amount DECIMAL(10,2) DEFAULT 0 COMMENT '团长金额',
    bonus_amount DECIMAL(10,2) DEFAULT 0 COMMENT '引流奖励',
    total_amount DECIMAL(10,2) DEFAULT 0 COMMENT '总金额',
    settled BOOLEAN DEFAULT FALSE COMMENT '是否已结算',
    settled_at DATETIME COMMENT '结算时间',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_task (task_id),
    INDEX idx_user (user_id),
    INDEX idx_status (status),
    INDEX idx_link_valid (link_valid),
    INDEX idx_validity_expire (validity_expire)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务提交表';

-- =====================================================
-- 7. 链接检查日志表
-- =====================================================
CREATE TABLE IF NOT EXISTS link_check_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    submission_id BIGINT NOT NULL COMMENT '提交记录ID',
    check_time DATETIME NOT NULL COMMENT '检查时间',
    is_valid BOOLEAN NOT NULL COMMENT '是否有效',
    likes_count INT DEFAULT 0 COMMENT '当时点赞数',
    response_code INT COMMENT 'HTTP响应码',
    error_message VARCHAR(500) COMMENT '错误信息',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_submission (submission_id),
    INDEX idx_check_time (check_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='链接检查日志表';

-- =====================================================
-- 8. 订单表
-- =====================================================
CREATE TABLE IF NOT EXISTS orders (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_no VARCHAR(50) NOT NULL UNIQUE COMMENT '订单号',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    merchant_id BIGINT NOT NULL COMMENT '商家ID',
    submission_id BIGINT COMMENT '关联提交ID',
    order_type ENUM('task_reward', 'commission', 'withdrawal') NOT NULL COMMENT '订单类型',
    amount DECIMAL(10,2) NOT NULL COMMENT '金额',
    status ENUM('pending', 'paid', 'cancelled') DEFAULT 'pending',
    remark VARCHAR(500) COMMENT '备注',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_merchant (merchant_id),
    INDEX idx_order_no (order_no)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单表';

-- =====================================================
-- 9. 提现记录表
-- =====================================================
CREATE TABLE IF NOT EXISTS withdrawals (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    amount DECIMAL(10,2) NOT NULL COMMENT '提现金额',
    payment_method VARCHAR(20) DEFAULT 'alipay' COMMENT '支付方式(alipay/wechat)',
    payment_account VARCHAR(100) COMMENT '收款账号',
    payment_qrcode VARCHAR(500) COMMENT '收款码图片',
    status ENUM('pending', 'approved', 'paid', 'rejected') DEFAULT 'pending',
    reject_reason VARCHAR(500) COMMENT '拒绝原因',
    reviewed_at DATETIME COMMENT '审核时间',
    reviewed_by BIGINT COMMENT '审核人ID',
    paid_at DATETIME COMMENT '打款时间',
    pay_proof VARCHAR(500) COMMENT '打款凭证',
    remark VARCHAR(500) COMMENT '备注',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='提现记录表';

-- =====================================================
-- 10. 分销佣金表
-- =====================================================
CREATE TABLE IF NOT EXISTS commissions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '获得佣金用户ID',
    from_user_id BIGINT NOT NULL COMMENT '来源用户ID',
    submission_id BIGINT COMMENT '关联提交ID',
    amount DECIMAL(10,2) NOT NULL COMMENT '佣金金额',
    commission_rate DECIMAL(5,2) COMMENT '抽成比例',
    status ENUM('pending', 'settled') DEFAULT 'pending',
    settled_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user (user_id),
    INDEX idx_from_user (from_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分销佣金表';

-- =====================================================
-- 11. 邀请关系表
-- =====================================================
CREATE TABLE IF NOT EXISTS invitations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    inviter_id BIGINT NOT NULL COMMENT '邀请人ID',
    invitee_id BIGINT NOT NULL COMMENT '被邀请人ID',
    invite_code VARCHAR(50) COMMENT '邀请码',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_invitee (invitee_id),
    INDEX idx_inviter (inviter_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邀请关系表';

-- =====================================================
-- 12. 系统配置表
-- =====================================================
CREATE TABLE IF NOT EXISTS system_config (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    merchant_id BIGINT COMMENT '商家ID(NULL表示全局)',
    config_key VARCHAR(100) NOT NULL COMMENT '配置键',
    config_value TEXT COMMENT '配置值',
    description VARCHAR(200) COMMENT '说明',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_merchant_key (merchant_id, config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置表';

-- =====================================================
-- 初始化数据
-- =====================================================

-- 插入总管理员账号
INSERT INTO users (username, password, nickname, role, status) VALUES 
('admin', '123456', '系统管理员', 'admin', 'active');

-- 插入测试商家账号
INSERT INTO users (username, password, nickname, role, status, expire_time) VALUES 
('merchant1', '123456', '测试商家', 'merchant', 'active', DATE_ADD(NOW(), INTERVAL 1 YEAR));

-- 插入测试用户账号
INSERT INTO users (username, password, nickname, role, status) VALUES 
('user1', '123456', '测试用户', 'user', 'active');

-- 插入平台配置 (小红书)
INSERT INTO platforms (name, code, icon, publish_url, status) VALUES 
('小红书', 'xiaohongshu', '/assets/icons/xiaohongshu.png', 'https://creator.xiaohongshu.com/publish/publish?source=official&from=menu&target=video&toShortText=true', 'active');

-- 插入默认系统配置
INSERT INTO system_config (config_key, config_value, description) VALUES 
('commission_rate', '10', '分销抽成比例(%)'),
('link_validity_days', '30', '链接有效期天数'),
('auto_check_interval', '6', '自动检查间隔(小时)'),
('min_withdraw_amount', '10', '最低提现金额');

SELECT '数据库初始化完成!' as message;
