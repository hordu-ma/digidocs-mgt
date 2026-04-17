package auth

type Claims struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
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
	PasswordHash string
	DisplayName  string
	Role         string
}
