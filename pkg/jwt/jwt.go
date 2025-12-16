package jwt

import (
	"context"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/jwt"
)

var (
	JwtMiddleware *jwt.HertzJWTMiddleware
	IdentityKey   = "im_user"
)

func InitJwt() {
	var err error
	JwtMiddleware, err = jwt.New(&jwt.HertzJWTMiddleware{
		Realm:         "test zone",
		Key:           []byte("secret key"),
		Timeout:       time.Hour,
		MaxRefresh:    time.Hour,
		TokenLookup:   "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		LoginResponse: func(ctx context.Context, c *app.RequestContext, code int, token string, expire time.Time) {
			c.JSON(http.StatusOK, utils.H{
				"code":    code,
				"token":   token,
				"expire":  expire.Format(time.RFC3339),
				"message": "success",
			})
		},
		// Authenticator: func(ctx context.Context, c *app.RequestContext) (interface{}, error) {
		// 	var loginStruct struct {
		// 		Account  string `form:"account" json:"account" query:"account" vd:"(len($) > 0 && len($) < 30); msg:'Illegal format'"`
		// 		Password string `form:"password" json:"password" query:"password" vd:"(len($) > 0 && len($) < 30); msg:'Illegal format'"`
		// 	}
		// 	if err := c.BindAndValidate(&loginStruct); err != nil {
		// 		return nil, err
		// 	}
		// 	users, err := mysql.CheckUser(loginStruct.Account, utils2.MD5(loginStruct.Password))
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	if len(users) == 0 {
		// 		return nil, errors.New("user already exists or wrong password")
		// 	}

		// 	return users[0], nil
		// },
		IdentityKey: IdentityKey,
		// IdentityHandler: func(ctx context.Context, c *app.RequestContext) interface{} {
		// 	claims := jwt.ExtractClaims(ctx, c)
		// 	return &model.User{
		// 		UserName: claims[IdentityKey].(string),
		// 	}
		// },
		// PayloadFunc: func(data interface{}) jwt.MapClaims {
		// 	if v, ok := data.(*model.User); ok {
		// 		return jwt.MapClaims{
		// 			IdentityKey: v.UserName,
		// 		}
		// 	}
		// 	return jwt.MapClaims{}
		// },
		HTTPStatusMessageFunc: func(e error, ctx context.Context, c *app.RequestContext) string {
			hlog.CtxErrorf(ctx, "jwt biz err = %+v", e.Error())
			return e.Error()
		},
		Unauthorized: func(ctx context.Context, c *app.RequestContext, code int, message string) {
			c.JSON(http.StatusOK, utils.H{
				"code":    code,
				"message": message,
			})
		},
	})
	if err != nil {
		panic(err)
	}
}
