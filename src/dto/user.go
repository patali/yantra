package dto

// UpdateThemeRequest represents the request to update user theme
type UpdateThemeRequest struct {
	Theme string `json:"theme" binding:"required"`
}

// UpdatePasswordRequest represents the request to update password
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}
