package request

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	// InviteToken is required when SELF_REGISTRATION=invite_only.
	InviteToken string `json:"invite_token"`
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Wechat      string `json:"wechat"`
}
