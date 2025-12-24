package model

type Auth struct {
	IsAuth     bool   `json:"is_auth"`     // 是否已登录
	UserID     int64  `json:"user_id"`     // 当前登陆用户id
	RemoteAddr string `json:"remote_addr"` // 当前登陆ip
}
