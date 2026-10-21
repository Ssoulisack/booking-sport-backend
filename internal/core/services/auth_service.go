package services

import (
	"backend/internal/adapters/secondary/redis"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
)

type AuthServiceImpl struct {
	userRepo  output.UserRepository
	cacheRepo redis.CacheRepository
}

func NewAuthService(
	userRepo output.UserRepository,
	cacheRepo redis.CacheRepository,
) input.AuthService {
	return &AuthServiceImpl{
		userRepo:  userRepo,
		cacheRepo: cacheRepo,
	}
}
