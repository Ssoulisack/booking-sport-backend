package dto

type UserDTO struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Status    bool   `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type UserJWTDTO struct {
	Access  string `json:"access,omitempty"`
	Refresh string `json:"refresh,omitempty"`
}
