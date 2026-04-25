package httpapi

import (
	"net/http"
)

// handleListNetworks lists networks in the project.
// @Summary      List networks
// @Tags         networks
// @Produce      json
// @Param        projectID  path  string  true  "Project UUID"
// @Success      200        {object}  docNetworksData
// @Failure      401        {object}  map[string]interface{}
// @Failure      403        {object}  map[string]interface{}
// @Failure      500        {object}  map[string]interface{}
// @Router       /v1/projects/{projectID}/networks [get]
// @Security     BearerAuth
func (s *Server) handleListNetworks(w http.ResponseWriter, r *http.Request) {
	projectID, ok := s.requireProjectAccess(w, r)
	if !ok {
		return
	}
	nets, err := s.projectSvc.ListNetworks(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list networks")
		return
	}
	writeData(w, http.StatusOK, nets)
}
