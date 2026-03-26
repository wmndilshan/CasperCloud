package httpapi

import (
	"net/http"

	"caspercloud/internal/auth"
	"caspercloud/internal/repository"
	"caspercloud/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", s.handleRegister)
			r.Post("/login", s.handleLogin)
		})

		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)
			r.Post("/projects", s.handleCreateProject)
			r.Get("/projects", s.handleListProjects)

			r.Route("/projects/{projectID}", func(r chi.Router) {
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

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return r
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
