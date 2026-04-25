package httpapi

import (
	"net/http"

	"caspercloud/internal/apitypes"
	"caspercloud/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// handleCreateSnapshot enqueues an internal QCOW2 snapshot.
// @Summary      Create instance snapshot
// @Tags         instances
// @Accept       json
// @Produce      json
// @Param        projectID   path  string                    true  "Project UUID"
// @Param        instanceID  path  string                    true  "Instance UUID"
// @Param        body        body  apitypes.CreateSnapshotRequest  true  "Snapshot label"
// @Success      202         {object}  docInstanceActionAccepted
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Failure      409         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID}/snapshots [post]
// @Security     BearerAuth
func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	var req apitypes.CreateSnapshotRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	res, err := s.instanceSvc.RequestSnapshotCreate(r.Context(), projectID, instanceID, req.Name)
	if err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusAccepted, res)
}

// handleListSnapshots lists snapshot metadata for an instance.
// @Summary      List instance snapshots
// @Tags         instances
// @Produce      json
// @Param        projectID   path  string  true  "Project UUID"
// @Param        instanceID  path  string  true  "Instance UUID"
// @Success      200         {object}  docSnapshotsData
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID}/snapshots [get]
// @Security     BearerAuth
func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	list, err := s.instanceSvc.ListSnapshots(r.Context(), projectID, instanceID)
	if err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusOK, list)
}

// handleRevertSnapshot enqueues revert to an internal snapshot.
// @Summary      Revert instance to snapshot
// @Tags         instances
// @Produce      json
// @Param        projectID   path  string  true  "Project UUID"
// @Param        instanceID  path  string  true  "Instance UUID"
// @Param        snapshotID  path  string  true  "Snapshot UUID"
// @Success      202         {object}  docInstanceActionAccepted
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Failure      409         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID}/snapshots/{snapshotID}/revert [post]
// @Security     BearerAuth
func (s *Server) handleRevertSnapshot(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	snapshotID, err := uuid.Parse(chi.URLParam(r, "snapshotID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid snapshot id")
		return
	}
	res, err := s.instanceSvc.RequestSnapshotRevert(r.Context(), projectID, instanceID, snapshotID)
	if err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusAccepted, res)
}
