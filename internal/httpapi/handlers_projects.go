package httpapi

import (
	"errors"
	"net/http"

	"caspercloud/internal/apitypes"
	"caspercloud/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// handleCreateProject creates a project for the authenticated user.
// @Summary      Create project
// @Description  Creates a new project and adds the caller as owner.
// @Tags         projects
// @Accept       json
// @Produce      json
// @Param        body  body      apitypes.CreateProjectRequest  true  "Project name"
// @Success      201   {object}  docProjectData
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /v1/projects [post]
// @Security     BearerAuth
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req apitypes.CreateProjectRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	project, err := s.projectSvc.CreateProject(r.Context(), userID, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create project")
		return
	}
	writeData(w, http.StatusCreated, project)
}

// handleListProjects lists projects the user belongs to.
// @Summary      List projects
// @Tags         projects
// @Produce      json
// @Success      200  {object}  docProjectsData
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /v1/projects [get]
// @Security     BearerAuth
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	projects, err := s.projectSvc.ListProjects(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list projects")
		return
	}
	writeData(w, http.StatusOK, projects)
}

// requireProjectAccess checks membership after route-level JWT project scope. Returns false if the handler should stop (response already written).
func (s *Server) requireProjectAccess(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return uuid.Nil, false
	}
	if err := s.projectSvc.EnsureProjectAccess(r.Context(), userID, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusForbidden, "project not accessible")
			return uuid.Nil, false
		}
		writeError(w, http.StatusInternalServerError, "could not verify project access")
		return uuid.Nil, false
	}
	return projectID, true
}
