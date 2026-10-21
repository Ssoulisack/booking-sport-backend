package rbac

import (
	"backend/internal/core/domain"
	"backend/pkg/jwt"
	"backend/pkg/response"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// WithRoles creates a middleware that checks the access token and that the
// user's role is one of allowedRoles. With no roles, any authenticated user passes.
func WithRoles(allowedRoles ...uint) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if !strings.HasPrefix(ctx.Get("Authorization"), "Bearer ") {
			return response.NewErrorErrMsgUnauthorized(ctx)
		}
		claims, err := jwt.GetOwnerAccessToken(ctx)
		if err != nil {
			return response.NewErrorErrMsgUnauthorized(ctx)
		}
		if !roleAllowed(uint(claims.RoleID), allowedRoles) {
			return response.NewErrorUnAuthorizeRole(ctx)
		}
		return ctx.Next()
	}
}

// roleAllowed checks if the user's role is in the list of allowed roles
func roleAllowed(userRole uint, allowedRoles []uint) bool {
	if len(allowedRoles) == 0 {
		return true
	}
	for _, role := range allowedRoles {
		if userRole == role {
			return true
		}
	}
	return false
}

// Predefined role permission sets
var (
	Authenticated = WithRoles()
	AdminOnly     = WithRoles(domain.RoleAdmin)
	StaffUp       = WithRoles(domain.RoleAdmin, domain.RoleStaff)
	ClientOnly    = WithRoles(domain.RoleClient)
)

// usage sample: router.Post("/", rbac.StaffUp, handler.Create)
