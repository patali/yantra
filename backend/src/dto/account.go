package dto

// CreateAccountRequest represents the request to create an account
type CreateAccountRequest struct {
	Name string `json:"name" binding:"required"`
}

// AddMemberRequest represents the request to add a member to an account
type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role"`
}

// AccountResponse represents the account response
type AccountResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Members   []MemberResponse `json:"members,omitempty"`
	CreatedAt string           `json:"createdAt"`
}

// MemberResponse represents the member response
type MemberResponse struct {
	UserID   string       `json:"userId"`
	Role     string       `json:"role"`
	User     UserResponse `json:"user,omitempty"`
	JoinedAt string       `json:"joinedAt"`
}
