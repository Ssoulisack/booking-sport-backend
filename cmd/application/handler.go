package application

import (
	"backend/internal/adapters/primary/http/handler"
	"backend/internal/core/mappers"
)

// handler.AuthHandler

type AppHandler struct {
	auth          handler.AuthHandler
	user          handler.UserHandler
}

func SetupHandlers(svc *AppServices) *AppHandler {
	classicMapper := mappers.NewClassicMapper[any, any]()
	mapperFactory := mappers.NewMapperFactory(classicMapper)
	return &AppHandler{
		auth:          handler.NewAuthHandler(svc.auth, mapperFactory),
		user:          handler.NewUserHandler(svc.user, mapperFactory),
	}
}
