package domain

import "context"

// UserRepository 仓储接口：领域层只定义“需要什么”，不关心“怎么存”
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	FindByUsername(ctx context.Context, username string) (*User, error)
}


