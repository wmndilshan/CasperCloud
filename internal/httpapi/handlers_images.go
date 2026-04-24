package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type imageRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=256"`
	SourceURL   string `json:"source_url" validate:"required,min=1,max=2048"`
	Description string `json:"description" validate:"max=4096"`
}

// handleCreateImage registers an image in the project.
// @Summary      Create image
// @Tags         images
// @Accept       json
// @Produce      json
// @Param        projectID  path      string           true  "Project UUID"
// @Param        body       body      imageRequest     true  "Image metadata"
// @Success      201        {object}  docImageData
// @Failure      400        {object}  map[string]interface{}
// @Failure      401        {object}  map[string]interface{}
// @Failure      403        {object}  map[string]interface{}
// @Failure      500        {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/images [post]
// @Security     BearerAuth
func (s *Server) handleCreateImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	var req imageRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	img, err := s.imageSvc.Create(r.Context(), projectID, req.Name, req.SourceURL, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create image")
		return
	}
	writeData(w, http.StatusCreated, img)
}

// handleListImages lists images in the project.
// @Summary      List images
// @Tags         images
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Success      200        {object}  docImagesData
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      500      {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/images [get]
// @Security     BearerAuth
func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	images, err := s.imageSvc.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list images")
		return
	}
	writeData(w, http.StatusOK, images)
}

// handleGetImage returns one image.
// @Summary      Get image
// @Tags         images
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Param        imageID    path  string  true  "Image UUID"
// @Success      200        {object}  docImageData
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      404      {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/images/{imageID} [get]
// @Security     BearerAuth
func (s *Server) handleGetImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
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
	writeData(w, http.StatusOK, img)
}

// handleUpdateImage updates image metadata.
// @Summary      Update image
// @Tags         images
// @Accept       json
// @Produce      json
// @Param        projectID  path  string        true  "Project UUID"
// @Param        imageID    path  string        true  "Image UUID"
// @Param        body       body  imageRequest  true  "Image metadata"
// @Success      200        {object}  docImageData
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      404      {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/images/{imageID} [put]
// @Security     BearerAuth
func (s *Server) handleUpdateImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	imageID, err := uuid.Parse(chi.URLParam(r, "imageID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid image id")
		return
	}
	var req imageRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	img, err := s.imageSvc.Update(r.Context(), projectID, imageID, req.Name, req.SourceURL, req.Description)
	if err != nil {
		status, msg := mapRepoError(err)
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusOK, img)
}

// handleDeleteImage deletes an image.
// @Summary      Delete image
// @Tags         images
// @Param        projectID  path  string  true  "Project UUID"
// @Param        imageID    path  string  true  "Image UUID"
// @Success      204        "No Content"
// @Failure      400        {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      404      {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/images/{imageID} [delete]
// @Security     BearerAuth
func (s *Server) handleDeleteImage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
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
