package auth

type Claims struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// UserRecord is the minimal user data fetched for authentication.
type UserRecord struct {
	ID           string
	PasswordHash string
	DisplayName  string
	Role         string
}
