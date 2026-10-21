package middleware

import (
	"backend/internal/core/domain"
	"backend/internal/core/port/input"
	"backend/pkg/jwt"
	"backend/pkg/response"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	localUser     = "user"
	localBusiness = "businessID"
	localRole     = "memberRole"
)

// Auth checks who is calling and what they may do. The token only proves identity:
// account status and business roles are read from the database on every request.
type Auth struct {
	access input.AccessService
}

func NewAuth(access input.AccessService) *Auth {
	return &Auth{access: access}
}

// authenticate returns (nil, nil) when no bearer token is sent
func (a *Auth) authenticate(c *fiber.Ctx) (*domain.User, error) {
	if !strings.HasPrefix(c.Get("Authorization"), "Bearer ") {
		return nil, nil
	}
	claims, err := jwt.GetOwnerAccessToken(c)
	if err != nil {
		return nil, response.NewError(http.StatusUnauthorized, response.ErrMsgInvalidAccessToken)
	}
	return a.access.Authenticate(c.UserContext(), uint(claims.ID), claims.IssuedAt)
}

// RequireUser lets only logged-in, enabled users through
func (a *Auth) RequireUser(c *fiber.Ctx) error {
	user, err := a.authenticate(c)
	if err != nil {
		return response.NewErrorResponses(c, err)
	}
	if user == nil {
		return response.NewErrorErrMsgUnauthorized(c)
	}
	c.Locals(localUser, user)
	return c.Next()
}

// OptionalUser loads the user when a token is sent; a bad token is still rejected
func (a *Auth) OptionalUser(c *fiber.Ctx) error {
	user, err := a.authenticate(c)
	if err != nil {
		return response.NewErrorResponses(c, err)
	}
	if user != nil {
		c.Locals(localUser, user)
	}
	return c.Next()
}

// RequireSuperAdmin must run after RequireUser
func (a *Auth) RequireSuperAdmin(c *fiber.Ctx) error {
	user := CurrentUser(c)
	if user == nil || !user.IsSuperAdmin() {
		return response.NewErrorUnAuthorizeRole(c)
	}
	return c.Next()
}

// Business admits members of the :businessId business whose role ranks at least minRole.
// A Super Admin may read (GET) any business for support. Must run after RequireUser.
func (a *Auth) Business(minRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("businessId")
		if err != nil || id < 1 {
			return response.NewErrorResponses(c, response.ErrorBadRequest("INVALID_ID"))
		}
		readOnly := c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead
		role, err := a.access.BusinessRole(c.UserContext(), CurrentUser(c), uint(id), minRole, readOnly)
		if err != nil {
			return response.NewErrorResponses(c, err)
		}
		c.Locals(localBusiness, uint(id))
		c.Locals(localRole, role)
		return c.Next()
	}
}

// CurrentUser is the user loaded by RequireUser / OptionalUser, or nil
func CurrentUser(c *fiber.Ctx) *domain.User {
	user, _ := c.Locals(localUser).(*domain.User)
	return user
}

// BusinessID is the business admitted by Business(), or 0
func BusinessID(c *fiber.Ctx) uint {
	id, _ := c.Locals(localBusiness).(uint)
	return id
}
