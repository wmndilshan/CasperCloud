package httpapi

import (
	"net/http"

	"caspercloud/internal/auth"
	"caspercloud/internal/repository"
	"caspercloud/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Server struct {
	authSvc     *service.AuthService
	projectSvc  *service.ProjectService
	imageSvc    *service.ImageService
	instanceSvc *service.InstanceService
	jwt         *auth.JWTManager
}

func NewServer(
	authSvc *service.AuthService,
	projectSvc *service.ProjectService,
	imageSvc *service.ImageService,
	instanceSvc *service.InstanceService,
	jwt *auth.JWTManager,
) *Server {
	return &Server{
		authSvc:     authSvc,
		projectSvc:  projectSvc,
		imageSvc:    imageSvc,
		instanceSvc: instanceSvc,
		jwt:         jwt,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealthz)

	r.Get("/docs", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/docs/", http.StatusFound)
	})
	r.Get("/docs/*", httpSwagger.WrapHandler)

	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", s.handleRegister)
			r.Post("/login", s.handleLogin)
		})

		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Post("/auth/switch-project", s.handleSwitchProject)
			r.Post("/projects", s.handleCreateProject)
			r.Get("/projects", s.handleListProjects)

			r.Route("/projects/{projectID}", func(r chi.Router) {
				r.Use(s.projectTokenMatchesURL)
				r.Route("/images", func(r chi.Router) {
					r.Post("/", s.handleCreateImage)
					r.Get("/", s.handleListImages)
					r.Get("/{imageID}", s.handleGetImage)
					r.Put("/{imageID}", s.handleUpdateImage)
					r.Delete("/{imageID}", s.handleDeleteImage)
				})
				r.Route("/instances", func(r chi.Router) {
					r.Post("/", s.handleCreateInstance)
					r.Get("/", s.handleListInstances)
					r.Get("/{instanceID}", s.handleGetInstance)
					r.Post("/{instanceID}/start", s.handleStartInstance)
					r.Post("/{instanceID}/stop", s.handleStopInstance)
					r.Post("/{instanceID}/reboot", s.handleRebootInstance)
					r.Delete("/{instanceID}", s.handleDeleteInstance)
				})
			})
		})
	})

	return r
}

// handleHealthz reports API readiness.
// @Summary      Health check
// @Description  Liveness endpoint; no authentication required.
// @Tags         system
// @Produce      json
// @Success      200  {object}  docHealthData
// @Router       /healthz [get]
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeData(w, http.StatusOK, map[string]string{"status": "ok"})
}

func mapRepoError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	if err == repository.ErrNotFound {
		return http.StatusNotFound, "resource not found"
	}
	return http.StatusInternalServerError, "internal server error"
}
