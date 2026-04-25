package apitypes

// RegisterRequest is the JSON body for POST /v1/auth/register.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

// LoginRequest is the JSON body for POST /v1/auth/login.
type LoginRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required"`
	ProjectID string `json:"project_id,omitempty" validate:"omitempty,uuid"`
}

// SwitchProjectRequest is the JSON body for POST /v1/auth/switch-project.
type SwitchProjectRequest struct {
	ProjectID string `json:"project_id" validate:"required,uuid"`
}
