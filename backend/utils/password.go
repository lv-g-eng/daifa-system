package utils

import "golang.org/x/crypto/bcrypt"

// IsHashed 判断字符串是否已是 bcrypt 哈希（$2a$/$2b$/$2y$ 开头、长度 60）
func IsHashed(s string) bool {
	if len(s) != 60 {
		return false
	}
	p := s[:4]
	return p == "$2a$" || p == "$2b$" || p == "$2y$"
}

// HashPassword 生成 bcrypt 哈希
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword 校验密码：
//   - 已哈希 → 走 bcrypt 比对
//   - 未哈希（历史明码数据）→ 明码比对，便于平滑迁移 / 登录时自动升级
func CheckPassword(stored, plain string) bool {
	if IsHashed(stored) {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil
	}
	return stored == plain
}
