package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type createProjectRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	project, err := s.projectSvc.CreateProject(r.Context(), userID, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create project")
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

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
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) requireProjectAccess(r *http.Request) (uuid.UUID, bool) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, false
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		return uuid.Nil, false
	}
	if err := s.projectSvc.EnsureProjectAccess(r.Context(), userID, projectID); err != nil {
		return uuid.Nil, false
	}
	return projectID, true
}
