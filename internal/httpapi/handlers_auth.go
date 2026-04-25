package httpapi

import (
	"net/http"
	"strings"

	"caspercloud/internal/apitypes"
	"caspercloud/internal/service"
	"github.com/google/uuid"
)

// handleRegister registers a new user; JWT is not scoped to a project.
// @Summary      Register
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      apitypes.RegisterRequest  true  "Credentials"
// @Success      201   {object}  docAuthData
// @Failure      400   {object}  map[string]interface{}
// @Router       /v1/auth/register [post]
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req apitypes.RegisterRequest
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
// @Param        body  body      LoginRequest  true  "Credentials"
// @Success      200   {object}  docAuthData
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /v1/auth/login [post]
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req apitypes.LoginRequest
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
// @Param        body  body      apitypes.SwitchProjectRequest  true  "Target project"
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
	var req apitypes.SwitchProjectRequest
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
