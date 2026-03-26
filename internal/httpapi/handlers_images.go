package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type imageRequest struct {
	Name        string `json:"name"`
	SourceURL   string `json:"source_url"`
	Description string `json:"description"`
}

func (s *Server) handleCreateImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
		return
	}
	var req imageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	img, err := s.imageSvc.Create(r.Context(), projectID, req.Name, req.SourceURL, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create image")
		return
	}
	writeJSON(w, http.StatusCreated, img)
}

func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
		return
	}
	images, err := s.imageSvc.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list images")
		return
	}
	writeJSON(w, http.StatusOK, images)
}

func (s *Server) handleGetImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
		return
	}
	imageID, err := uuid.Parse(chi.URLParam(r, "imageID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid image id")
		return
	}
	img, err := s.imageSvc.Get(r.Context(), projectID, imageID)
	if err != nil {
		status, msg := mapRepoError(err)
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, img)
}

func (s *Server) handleUpdateImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
		return
	}
	imageID, err := uuid.Parse(chi.URLParam(r, "imageID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid image id")
		return
	}
	var req imageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	img, err := s.imageSvc.Update(r.Context(), projectID, imageID, req.Name, req.SourceURL, req.Description)
	if err != nil {
		status, msg := mapRepoError(err)
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, img)
}

func (s *Server) handleDeleteImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
		return
	}
	imageID, err := uuid.Parse(chi.URLParam(r, "imageID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid image id")
		return
	}
	if err := s.imageSvc.Delete(r.Context(), projectID, imageID); err != nil {
		status, msg := mapRepoError(err)
		writeError(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
