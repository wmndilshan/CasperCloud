package httpapi

import (
	"caspercloud/internal/repository"
	"caspercloud/internal/service"
)

// The following types exist only for Swag / OpenAPI response schemas (JSON envelope {"data": ...}).

type docAuthData struct {
	Data service.AuthResponse `json:"data"`
}

type docProjectData struct {
	Data repository.Project `json:"data"`
}

type docProjectsData struct {
	Data []repository.Project `json:"data"`
}

type docImageData struct {
	Data repository.Image `json:"data"`
}

type docImagesData struct {
	Data []repository.Image `json:"data"`
}

type docInstanceData struct {
	Data repository.Instance `json:"data"`
}

type docInstancesData struct {
	Data []repository.Instance `json:"data"`
}

type docCreateInstanceAccepted struct {
	Data service.CreateInstanceResult `json:"data"`
}

type docStatusOK struct {
	Data map[string]string `json:"data"`
}

type docHealthData struct {
	Data map[string]string `json:"data"`
}
