package httpapi

import (
	"net/http"

	"caspercloud/internal/apitypes"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// handleAllocateFloatingIP claims a public address from the pool for the project.
// @Summary      Allocate floating IP
// @Tags         floating-ips
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Success      201  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/floating-ips [post]
// @Security     BearerAuth
func (s *Server) handleAllocateFloatingIP(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	fip, err := s.floatingIPSvc.Allocate(r.Context(), projectID)
	if err != nil {
		status, msg := mapRepoError(err)
		if status == http.StatusInternalServerError {
			msg = err.Error()
		}
		writeError(w, status, msg)
		return
	}
	writeData(w, http.StatusCreated, fip)
}

// handleListFloatingIPs lists floating IPs for the project.
// @Summary      List floating IPs
// @Tags         floating-ips
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Success      200  {array}   object
// @Router       /v1/projects/{projectID}/floating-ips [get]
// @Security     BearerAuth
func (s *Server) handleListFloatingIPs(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	list, err := s.floatingIPSvc.List(r.Context(), projectID)
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

// handleAssociateFloatingIP binds a floating IP to a running instance and enqueues NAT setup.
// @Summary      Associate floating IP
// @Tags         floating-ips
// @Accept       json
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Param        floatingIPID  path  string  true  "Floating IP UUID"
// @Param        body  body  apitypes.AssociateFloatingIPRequest  true  "Target instance"
// @Success      202  {object}  docInstanceActionAccepted
// @Router       /v1/projects/{projectID}/floating-ips/{floatingIPID}/associate [post]
// @Security     BearerAuth
func (s *Server) handleAssociateFloatingIP(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	floatingIPID, err := uuid.Parse(chi.URLParam(r, "floatingIPID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid floating ip id")
		return
	}
	var req apitypes.AssociateFloatingIPRequest
	if err := decodeAndValidate(r, &req); err != nil {
		respondInvalidRequest(w, err)
		return
	}
	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid instance id")
		return
	}
	res, err := s.floatingIPSvc.RequestAssociate(r.Context(), projectID, floatingIPID, instanceID)
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

// handleDisassociateFloatingIP releases the instance binding and enqueues NAT teardown.
// @Summary      Disassociate floating IP
// @Tags         floating-ips
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Param        floatingIPID  path  string  true  "Floating IP UUID"
// @Success      202  {object}  docInstanceActionAccepted
// @Router       /v1/projects/{projectID}/floating-ips/{floatingIPID}/disassociate [post]
// @Security     BearerAuth
func (s *Server) handleDisassociateFloatingIP(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	floatingIPID, err := uuid.Parse(chi.URLParam(r, "floatingIPID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid floating ip id")
		return
	}
	res, err := s.floatingIPSvc.RequestDisassociate(r.Context(), projectID, floatingIPID)
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
