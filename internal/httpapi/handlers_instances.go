package httpapi

import (
	"net/http"

	"caspercloud/internal/apitypes"
	"caspercloud/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// handleCreateInstance enqueues instance creation.
// @Summary      Create instance
// @Tags         instances
// @Accept       json
// @Produce      json
// @Param        projectID  path      string                 true  "Project UUID"
// @Param        body       body      apitypes.CreateInstanceRequest  true  "Instance spec"
// @Success      202        {object}  docCreateInstanceAccepted
// @Failure      400        {object}  map[string]interface{}
// @Failure      401        {object}  map[string]interface{}
// @Failure      403        {object}  map[string]interface{}
// @Failure      404        {object}  map[string]interface{}
// @Failure      500        {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances [post]
// @Security     BearerAuth
func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	var req apitypes.CreateInstanceRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	imageID := uuid.MustParse(req.ImageID)
	in := service.CreateInstanceInput{
		Name:         req.Name,
		ImageID:      imageID,
		Hostname:     req.Hostname,
		Username:     req.Username,
		SSHPublicKey: req.SSHPublicKey,
		Packages:     req.Packages,
		RunCommands:  req.RunCommands,
	}
	if req.NetworkID != "" {
		nid, perr := uuid.Parse(req.NetworkID)
		if perr != nil {
			writeError(w, http.StatusBadRequest, "invalid network id")
			return
		}
		in.NetworkID = &nid
	}
	res, err := s.instanceSvc.CreateAsync(r.Context(), projectID, in)
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

// handleGetInstance returns instance details.
// @Summary      Get instance
// @Tags         instances
// @Produce      json
// @Param        projectID   path  string  true  "Project UUID"
// @Param        instanceID  path  string  true  "Instance UUID"
// @Success      200         {object}  docInstanceData
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID} [get]
// @Security     BearerAuth
func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	inst, err := s.instanceSvc.Get(r.Context(), projectID, instanceID)
	if err != nil {
		status, msg := mapRepoError(err)
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusOK, inst)
}

// handleGetInstanceStats returns the latest libvirt-backed metrics snapshot (requires Redis populated by the worker).
// @Summary      Instance hardware stats
// @Tags         instances
// @Produce      json
// @Param        projectID   path  string  true  "Project UUID"
// @Param        instanceID  path  string  true  "Instance UUID"
// @Success      200         {object}  docInstanceStatsData
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Failure      503         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID}/stats [get]
// @Security     BearerAuth
func (s *Server) handleGetInstanceStats(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	stats, err := s.instanceSvc.GetInstanceStats(r.Context(), projectID, instanceID)
	if err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusOK, stats)
}

// handleListInstances lists instances in the project.
// @Summary      List instances
// @Tags         instances
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Success      200        {object}  docInstancesData
// @Failure      401        {object}  map[string]interface{}
// @Failure      403        {object}  map[string]interface{}
// @Failure      500        {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances [get]
// @Security     BearerAuth
func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instances, err := s.instanceSvc.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list instances")
		return
	}
	writeData(w, http.StatusOK, instances)
}

// handleInstanceActions enqueues start, stop, reboot, or destroy for the worker.
// @Summary      Instance lifecycle action
// @Tags         instances
// @Accept       json
// @Produce      json
// @Param        projectID   path  string                 true  "Project UUID"
// @Param        instanceID  path  string                 true  "Instance UUID"
// @Param        body        body  apitypes.InstanceActionRequest    true  "Action"
// @Success      202         {object}  docInstanceActionAccepted
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Failure      500         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID}/actions [post]
// @Security     BearerAuth
func (s *Server) handleInstanceActions(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	var req apitypes.InstanceActionRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	res, err := s.instanceSvc.RequestInstanceAction(r.Context(), projectID, instanceID, req.Action)
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

// handleDeleteInstance enqueues destroy (same as action "destroy") for backward-compatible clients.
// @Summary      Delete instance (async destroy)
// @Tags         instances
// @Param        projectID   path  string  true  "Project UUID"
// @Param        instanceID  path  string  true  "Instance UUID"
// @Success      202         {object}  docInstanceActionAccepted
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Failure      500         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID} [delete]
// @Security     BearerAuth
func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	res, err := s.instanceSvc.RequestInstanceAction(r.Context(), projectID, instanceID, "destroy")
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

// handleAttachVolume hot-attaches a project volume to a running instance.
// @Summary      Attach volume to instance
// @Tags         instances
// @Accept       json
// @Produce      json
// @Param        projectID   path  string                      true  "Project UUID"
// @Param        instanceID  path  string                      true  "Instance UUID"
// @Param        body        body  apitypes.InstanceVolumeAttachRequest  true  "Volume to attach"
// @Success      204         "No Content"
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Failure      409         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID}/attach [post]
// @Security     BearerAuth
func (s *Server) handleAttachVolume(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	var attachReq apitypes.InstanceVolumeAttachRequest
	if err := decodeAndValidate(r, &attachReq); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	volumeID, perr := uuid.Parse(attachReq.VolumeID)
	if perr != nil {
		writeError(w, http.StatusBadRequest, "invalid volume id")
		return
	}
	if err := s.volumeSvc.AttachVolume(r.Context(), projectID, instanceID, volumeID); err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleDetachVolume detaches a volume from an instance.
// @Summary      Detach volume from instance
// @Tags         instances
// @Accept       json
// @Produce      json
// @Param        projectID   path  string                      true  "Project UUID"
// @Param        instanceID  path  string                      true  "Instance UUID"
// @Param        body        body  apitypes.InstanceVolumeAttachRequest  true  "Volume to detach"
// @Success      204         "No Content"
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID}/detach [post]
// @Security     BearerAuth
func (s *Server) handleDetachVolume(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	var detachReq apitypes.InstanceVolumeAttachRequest
	if err := decodeAndValidate(r, &detachReq); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	volumeID, perr := uuid.Parse(detachReq.VolumeID)
	if perr != nil {
		writeError(w, http.StatusBadRequest, "invalid volume id")
		return
	}
	if err := s.volumeSvc.DetachVolume(r.Context(), projectID, instanceID, volumeID); err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
