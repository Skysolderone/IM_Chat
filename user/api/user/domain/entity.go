package domain

// User 领域实体（不包含 ORM 细节）
type User struct {
	ID           uint
	Username     string
	PasswordHash string
}
