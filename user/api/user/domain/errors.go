package domain

import "errors"

var (
	// ErrUserAlreadyExists 用户名已存在
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("user not found")
)
