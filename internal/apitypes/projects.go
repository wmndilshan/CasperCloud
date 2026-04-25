package apitypes

// CreateProjectRequest is the JSON body for POST /v1/projects.
type CreateProjectRequest struct {
	Name string `json:"name" validate:"required,min=1,max=128"`
}
