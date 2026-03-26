package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"caspercloud/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type createInstanceRequest struct {
	Name         string   `json:"name"`
	ImageID      string   `json:"image_id"`
	Hostname     string   `json:"hostname"`
	Username     string   `json:"username"`
	SSHPublicKey string   `json:"ssh_public_key"`
	Packages     []string `json:"packages"`
	RunCommands  []string `json:"run_commands"`
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
		return
	}
	var req createInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	imageID, err := uuid.Parse(req.ImageID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid image id")
		return
	}
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
	writeJSON(w, http.StatusAccepted, res)
}

func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
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
	writeJSON(w, http.StatusOK, inst)
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
		return
	}
	instances, err := s.instanceSvc.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list instances")
		return
	}
	writeJSON(w, http.StatusOK, instances)
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

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
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

func (s *Server) handleInstanceAction(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID, uuid.UUID) error) {
	projectID, ok := s.requireProjectAccess(r)
	if !ok {
		writeError(w, http.StatusForbidden, "project not accessible")
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
