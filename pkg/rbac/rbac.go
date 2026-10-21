package rbac

import (
	"backend/internal/config"
	"backend/pkg/jwt"
	"backend/pkg/logger"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	jwtktr "github.com/kataras/jwt"
)

// RolePermission defines the allowed roles for specific routes
type RolePermission struct {
	Roles []string
}

var logs = logger.NewLogger()

// WithRoles creates a middleware that checks if the user has the required roles
func WithRoles(allowedRoles ...string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// First, verify access token
		err := jwt.AccessToken(ctx)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  false,
				"message": "Unauthorized",
			})
		}

		// Extract user role from token
		userRole, err := extractUserRole(ctx)
		if err != nil {
			logs.Error(fmt.Sprintf("Role extraction failed: %v", err), nil)
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  false,
				"message": "Unauthorized",
			})
		}

		// Log the user role for debugging
		// fmt.Println("Role from header:", userRole)
		// fmt.Println("Role from header:", allowedRoles)

		// Check if user's role is in the allowed roles
		if !roleAllowed(userRole, allowedRoles) {
			logs.Error(fmt.Sprintf("Role access denied. User role: %s, Allowed roles: %v", userRole, allowedRoles), nil)
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  false,
				"message": "Forbidden: Insufficient permissions",
			})
		}
		// Important: Always call Next() to pass control to the next handler
		return nil
	}
}

// extractUserRole extracts the user's role from the access token
func extractUserRole(ctx *fiber.Ctx) (string, error) {
	auth := ctx.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("no authorization token")
	}

	jwtFromHeader := strings.TrimSpace(auth[7:])
	_, decrypt, _ := jwtktr.GCM([]byte(config.JWTGlobal.Secret), nil)
	verifiedToken, err := jwtktr.VerifyEncrypted(jwtktr.HS256, []byte(config.JWTGlobal.Secret), decrypt, []byte(jwtFromHeader))
	if err != nil {
		return "", err
	}

	var claims jwt.ClaimsToken
	err = verifiedToken.Claims(&claims)
	if err != nil {
		return "", err
	}

	// Log the extracted role for debugging
	// logs.Info(fmt.Sprintf("Extracted role: %s", claims.Role))

	return fmt.Sprintf("%d", claims.RoleID), nil
}

// roleAllowed checks if the user's role is in the list of allowed roles
func roleAllowed(userRole string, allowedRoles []string) bool {
	// If no roles are specified, allow access
	if len(allowedRoles) == 0 {
		return true
	}

	// Check if user's role is in the allowed roles
	for _, role := range allowedRoles {
		if strings.EqualFold(userRole, role) {
			return true
		}
	}

	return false
}

// Predefined role permission sets
var (
	UserOnly  = WithRoles("USER")
	AdminOnl = WithRoles("ADMIN")

	// Combinations
	AdminOrUser = WithRoles("ADMIN", "USER")
)

// useage sample: roleRoute.Post("/", middleware.WithRoles(middleware.RoleAdmin, middleware.RoleManager), roleController.Create)
