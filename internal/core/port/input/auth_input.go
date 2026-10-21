package input

import (
	"backend/internal/core/dto"
	"context"
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterDTO) (*dto.AuthResponseDTO, error)
	Login(ctx context.Context, req dto.LoginDTO) (*dto.AuthResponseDTO, error)
	Refresh(ctx context.Context, userID uint) (*dto.AuthResponseDTO, error)
	Me(ctx context.Context, userID uint) (*dto.UserDTO, error)
}
