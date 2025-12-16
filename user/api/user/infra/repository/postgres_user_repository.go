package repository

import (
	"context"
	"errors"
	"time"

	"wsim/user/api/user/domain"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type UserModel struct {
	gorm.Model
	Username     string    `gorm:"type:varchar(64);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	LastLoginAt  time.Time `gorm:"type:timestamp;not null"`
	LastLoginIP  string    `gorm:"type:varchar(45);not null"`
}

func (UserModel) TableName() string { return "users" }

type PostgresUserRepository struct {
	db *gorm.DB
}

func NewPostgresUserRepository(db *gorm.DB) (*PostgresUserRepository, error) {
	if err := db.AutoMigrate(&UserModel{}); err != nil {
		return nil, err
	}
	// 性能：username 精确查找 + 软删除条件（deleted_at IS NULL）
	// 注意：Gorm 的 uniqueIndex 不会创建“部分索引”，这里手动补一个（只用于查询加速）
	_ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_username_not_deleted ON users (username) WHERE deleted_at IS NULL;`).Error
	return &PostgresUserRepository{db: db}, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, u *domain.User) error {
	m := &UserModel{
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
	}
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return err
	}
	m.LastLoginAt = time.Now()
	m.LastLoginIP = ctx.Value("x-forwarded-for").(string)
	err = r.db.WithContext(ctx).Save(m).Error
	if err != nil {
		return err
	}
	u.ID = m.ID
	return nil
}

func (r *PostgresUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var m UserModel
	// 用 Find + Limit(1) 避免 ErrRecordNotFound 被当成 error 打日志；
	// 同时只取需要的字段，避免 SELECT *。
	tx := r.db.WithContext(ctx).
		Select("id", "username", "password_hash").
		Where("username = ?", username).
		Limit(1).
		Find(&m)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, domain.ErrUserNotFound
	}
	return &domain.User{
		ID:           m.ID,
		Username:     m.Username,
		PasswordHash: m.PasswordHash,
	}, nil
}
