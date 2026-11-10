package auth

type Claims struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	// IssuedAt is the token's iat (unix seconds); set by TokenService.Parse,
	// used to reject tokens issued before the last password change.
	IssuedAt int64 `json:"-"`
}

type UserProfile struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	Email       string  `json:"email,omitempty"`
	Phone       string  `json:"phone,omitempty"`
	Wechat      string  `json:"wechat,omitempty"`
	Status      string  `json:"status"`
	LastLoginAt *string `json:"last_login_at"`
}

type ProfileUpdateInput struct {
	DisplayName string
	Email       string
	Phone       string
	Wechat      string
}

// UserRecord is the minimal user data fetched for authentication.
type UserRecord struct {
	ID           string
	Username     string
	PasswordHash string
	DisplayName  string
	Role         string
}

// RegisterUserInput is the persistence payload for self-service registration.
// Username is intentionally omitted: self-registered accounts have no username.
type RegisterUserInput struct {
	Email        string
	Phone        string
	PasswordHash string
	DisplayName  string
}
