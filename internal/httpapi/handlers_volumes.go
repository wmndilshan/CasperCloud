package httpapi

import (
	"net/http"

	"caspercloud/internal/apitypes"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// handleCreateVolume provisions a sparse qcow2 volume in the project.
// @Summary      Create volume
// @Tags         volumes
// @Accept       json
// @Produce      json
// @Param        projectID  path  string               true  "Project UUID"
// @Param        body       body  apitypes.CreateVolumeRequest  true  "Volume spec"
// @Success      201        {object}  docVolumeData
// @Failure      400      {object}  map[string]interface{}
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      409      {object}  map[string]interface{}
// @Failure      500      {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/volumes [post]
// @Security     BearerAuth
func (s *Server) handleCreateVolume(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	var req apitypes.CreateVolumeRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	vol, err := s.volumeSvc.CreateVolume(r.Context(), projectID, req.Name, req.SizeGB)
	if err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusCreated, vol)
}

// handleListVolumes lists volumes in the project.
// @Summary      List volumes
// @Tags         volumes
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Success      200        {object}  docVolumesData
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      500      {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/volumes [get]
// @Security     BearerAuth
func (s *Server) handleListVolumes(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	vols, err := s.volumeSvc.ListVolumes(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list volumes")
		return
	}
	writeData(w, http.StatusOK, vols)
}

// handleGetVolume returns one volume.
// @Summary      Get volume
// @Tags         volumes
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Param        volumeID   path  string  true  "Volume UUID"
// @Success      200        {object}  docVolumeData
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      404      {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/volumes/{volumeID} [get]
// @Security     BearerAuth
func (s *Server) handleGetVolume(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	volumeID, err := uuid.Parse(chi.URLParam(r, "volumeID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid volume id")
		return
	}
	vol, err := s.volumeSvc.GetVolume(r.Context(), projectID, volumeID)
	if err != nil {
		status, msg := mapRepoError(err)
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusOK, vol)
}

// handleDeleteVolume deletes a detached volume and its qcow2 file.
// @Summary      Delete volume
// @Tags         volumes
// @Param        projectID  path  string  true  "Project UUID"
// @Param        volumeID   path  string  true  "Volume UUID"
// @Success      204        "No Content"
// @Failure      401      {object}  map[string]interface{}
// @Failure      403      {object}  map[string]interface{}
// @Failure      404      {object}  map[string]interface{}
// @Failure      409      {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/volumes/{volumeID} [delete]
// @Security     BearerAuth
func (s *Server) handleDeleteVolume(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	volumeID, err := uuid.Parse(chi.URLParam(r, "volumeID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid volume id")
		return
	}
	if err := s.volumeSvc.DeleteVolume(r.Context(), projectID, volumeID); err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
