package routes

import (
	"log"

	"wsim/pkg/postgresql"
	"wsim/user/api/user/handler"
	"wsim/user/api/user/infra/password"
	"wsim/user/api/user/infra/repository"
	"wsim/user/api/user/infra/token"
	"wsim/user/api/user/usecase"

	"github.com/cloudwego/hertz/pkg/app/server"
)

func InitRouter(h *server.Hertz) {
	repo, err := repository.NewPostgresUserRepository(postgresql.GetDB())
	if err != nil {
		log.Fatalf("init user repository failed: %v", err)
	}
	authSvc := usecase.NewAuthService(
		repo,
		password.NewBcryptHasher(0),
		token.NewJWTGenerator(),
	)
	authHandler := handler.NewAuthHandler(authSvc)

	h.POST("/user/login", authHandler.Login)
	h.POST("/user/register", authHandler.Register)
	
}
