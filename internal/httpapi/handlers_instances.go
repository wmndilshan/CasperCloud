package httpapi

import (
	"context"
	"net/http"

	"caspercloud/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type createInstanceRequest struct {
	Name         string   `json:"name" validate:"required,min=1,max=128"`
	ImageID      string   `json:"image_id" validate:"required,uuid"`
	Hostname     string   `json:"hostname" validate:"required,min=1,max=253"`
	Username     string   `json:"username" validate:"required,min=1,max=32"`
	SSHPublicKey string   `json:"ssh_public_key" validate:"required,min=1,max=8192"`
	Packages     []string `json:"packages" validate:"max=64,dive,max=128"`
	RunCommands  []string `json:"run_commands" validate:"max=32,dive,max=2048"`
}

// handleCreateInstance enqueues instance creation.
// @Summary      Create instance
// @Tags         instances
// @Accept       json
// @Produce      json
// @Param        projectID  path      string                 true  "Project UUID"
// @Param        body       body      createInstanceRequest  true  "Instance spec"
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
	var req createInstanceRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	imageID := uuid.MustParse(req.ImageID)
	res, err := s.instanceSvc.CreateAsync(r.Context(), projectID, service.CreateInstanceInput{
		Name:         req.Name,
		ImageID:      imageID,
		Hostname:     req.Hostname,
		Username:     req.Username,
		SSHPublicKey: req.SSHPublicKey,
		Packages:     req.Packages,
		RunCommands:  req.RunCommands,
	})
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

func (s *Server) handleStartInstance(w http.ResponseWriter, r *http.Request) {
	s.handleInstanceAction(w, r, s.instanceSvc.Start)
}

func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request) {
	s.handleInstanceAction(w, r, s.instanceSvc.Stop)
}

func (s *Server) handleRebootInstance(w http.ResponseWriter, r *http.Request) {
	s.handleInstanceAction(w, r, s.instanceSvc.Reboot)
}

// handleDeleteInstance deletes an instance.
// @Summary      Delete instance
// @Tags         instances
// @Param        projectID   path  string  true  "Project UUID"
// @Param        instanceID  path  string  true  "Instance UUID"
// @Success      204         "No Content"
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
	if err := s.instanceSvc.Delete(r.Context(), projectID, instanceID); err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleInstanceAction runs start/stop/reboot.
// @Summary      Instance lifecycle action
// @Tags         instances
// @Produce      json
// @Param        projectID   path  string  true  "Project UUID"
// @Param        instanceID  path  string  true  "Instance UUID"
// @Success      200         {object}  docStatusOK
// @Failure      400         {object}  map[string]interface{}
// @Failure      401         {object}  map[string]interface{}
// @Failure      403         {object}  map[string]interface{}
// @Failure      404         {object}  map[string]interface{}
// @Failure      500         {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/instances/{instanceID}/start [post]
// @Router       /v1/projects/{projectID}/instances/{instanceID}/stop [post]
// @Router       /v1/projects/{projectID}/instances/{instanceID}/reboot [post]
// @Security     BearerAuth
func (s *Server) handleInstanceAction(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID, uuid.UUID) error) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	instanceID, err := uuid.Parse(chi.URLParam(r, "instanceID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	if err := fn(r.Context(), projectID, instanceID); err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusOK, map[string]string{"status": "ok"})
}
