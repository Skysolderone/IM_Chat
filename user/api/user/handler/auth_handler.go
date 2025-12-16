package handler

import (
	"context"
	"errors"
	"net/http"

	"wsim/user/api/user/domain"
	"wsim/user/api/user/dto"
	"wsim/user/api/user/usecase"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

type AuthHandler struct {
	auth *usecase.AuthService
}

func NewAuthHandler(auth *usecase.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(ctx context.Context, c *app.RequestContext) {
	var req dto.RegisterRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	ip := c.ClientIP()
	ctx = context.WithValue(ctx, "x-forwarded-for", ip)
	res, err := h.auth.Register(ctx, req.Username, req.Password)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.RegisterResponse{ID: res.UserID, Token: res.Token})
}

func (h *AuthHandler) Login(ctx context.Context, c *app.RequestContext) {
	var req dto.LoginRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}
	res, err := h.auth.Login(ctx, req.Username, req.Password)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.LoginResponse{ID: res.UserID, Token: res.Token})
}

func (h *AuthHandler) writeErr(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, usecase.ErrBadRequest):
		c.JSON(http.StatusBadRequest, utils.H{"error": "bad request"})
	case errors.Is(err, domain.ErrUserAlreadyExists):
		c.JSON(http.StatusConflict, utils.H{"error": "user already exists"})
	case errors.Is(err, usecase.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, utils.H{"error": "invalid credentials"})
	default:
		c.JSON(http.StatusInternalServerError, utils.H{"error": err.Error()})
	}
}
