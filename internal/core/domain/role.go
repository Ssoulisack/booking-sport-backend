package domain

// Role IDs are stored in the JWT (role_id claim) and checked by pkg/rbac
const (
	RoleAdmin  uint = 1
	RoleStaff  uint = 2
	RoleClient uint = 3
)

// Role is the gorm model for the roles table
type Role struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"type:varchar(50);uniqueIndex;not null"`
}
