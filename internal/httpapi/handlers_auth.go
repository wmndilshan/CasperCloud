package httpapi

import (
	"net/http"
	"strings"

	"caspercloud/internal/service"
	"github.com/google/uuid"
)

type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type loginRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required"`
	ProjectID string `json:"project_id,omitempty" validate:"omitempty,uuid"`
}

type switchProjectRequest struct {
	ProjectID string `json:"project_id" validate:"required,uuid"`
}

// handleRegister registers a new user; JWT is not scoped to a project.
// @Summary      Register
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      registerRequest  true  "Credentials"
// @Success      201   {object}  docAuthData
// @Failure      400   {object}  map[string]interface{}
// @Router       /v1/auth/register [post]
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	resp, err := s.authSvc.Register(r.Context(), strings.TrimSpace(req.Email), req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeData(w, http.StatusCreated, resp)
}

// handleLogin returns a JWT; optional project_id scopes the token to that project.
// @Summary      Login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Credentials"
// @Success      200   {object}  docAuthData
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /v1/auth/login [post]
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	in := service.LoginInput{
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
	}
	if strings.TrimSpace(req.ProjectID) != "" {
		pid := uuid.MustParse(req.ProjectID)
		in.ProjectID = &pid
	}
	resp, err := s.authSvc.Login(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeData(w, http.StatusOK, resp)
}

// handleSwitchProject issues a new JWT scoped to the given project.
// @Summary      Switch active project
// @Description  Returns a new token whose active_project_id matches the requested project (must be a member).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      switchProjectRequest  true  "Target project"
// @Success      200   {object}  docAuthData
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      403   {object}  map[string]interface{}
// @Router       /v1/auth/switch-project [post]
// @Security     BearerAuth
func (s *Server) handleSwitchProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req switchProjectRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	projectID := uuid.MustParse(req.ProjectID)
	resp, err := s.authSvc.SwitchProject(r.Context(), userID, projectID)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	writeData(w, http.StatusOK, resp)
}
