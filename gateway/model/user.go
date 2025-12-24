package model

import "github.com/cloudwego/netpoll"

type User struct {
	UserID int64              `json:"user_id"`
	Conn   netpoll.Connection `json:"conn"`
	IsAuth bool               `json:"is_auth"`
}

var Users map[int64]*User

func NewUsers() {
	Users = make(map[int64]*User)
}

func (u *User) Get(userID int64) *User {
	return Users[userID]
}

func (u *User) Set(userID int64, user *User) {
	Users[userID] = user
}

func (u *User) Delete(userID int64) {
	delete(Users, userID)
}

// 获取用户是否登陆状态
func (u *User) AuthStatus(userID int64) bool {
	if user, ok := Users[userID]; ok {
		return user.IsAuth
	}
	return false
}

// 设置用户登陆状态
func (u *User) SetAuth(userID int64, isAuth bool) {
	if user, ok := Users[userID]; ok {
		user.IsAuth = isAuth
	} else {
		Users[userID] = &User{
			UserID: userID,
			IsAuth: isAuth,
		}
	}
}

func (u *User) GetConn(userID int64) netpoll.Connection {
	if user, ok := Users[userID]; ok {
		return user.Conn
	}
	return nil
}
