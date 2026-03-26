# CasperCloud Milestone 1

Mini AWS-like control plane (Milestone 1) with:
- Go backend
- PostgreSQL source-of-truth state
- JWT auth
- project/tenant isolation
- image metadata CRUD
- instance CRUD + lifecycle actions
- async worker (RabbitMQ)
- libvirt integration adapter skeleton (`virsh` based)
- cloud-init user-data generation
- OpenAPI spec
- SQL migrations
- Docker Compose

## Folder Structure

```text
.
├── api
│   └── openapi.yaml
├── cmd
│   ├── api
│   │   └── main.go
│   └── worker
│       └── main.go
├── docs
│   └── sample_requests.http
├── internal
│   ├── auth
│   │   ├── jwt.go
│   │   └── password.go
│   ├── cloudinit
│   │   └── generator.go
│   ├── config
│   │   └── config.go
│   ├── db
│   │   └── postgres.go
│   ├── httpapi
│   │   ├── handlers_auth.go
│   │   ├── handlers_images.go
│   │   ├── handlers_instances.go
│   │   ├── handlers_projects.go
│   │   ├── middleware.go
│   │   ├── responses.go
│   │   └── server.go
│   ├── libvirt
│   │   ├── adapter.go
│   │   └── virsh_adapter.go
│   ├── queue
│   │   └── rabbitmq.go
│   ├── repository
│   │   ├── models.go
│   │   └── repository.go
│   ├── service
│   │   ├── auth_service.go
│   │   ├── image_service.go
│   │   ├── instance_service.go
│   │   └── project_service.go
│   └── worker
│       └── worker.go
├── migrations
│   └── 0001_init.sql
├── .env.example
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

## Prerequisites

For local host VM lifecycle (outside containers), use Ubuntu host with:
- KVM/libvirt installed and running
- `virsh` available
- network bridge setup as needed

## Quick Start (Docker Compose)

From project root:

```powershell
Set-Location "d:\Projects\casperCloud"
docker compose up -d --build
```

Services:
- API: `http://localhost:8080`
- RabbitMQ UI: `http://localhost:15672` (`guest` / `guest`)
- PostgreSQL: `localhost:5432`

Migrations are auto-applied from `migrations/` by Postgres init on first DB startup.

Health check:

```powershell
Invoke-RestMethod -Method Get -Uri "http://localhost:8080/healthz"
```

## Local Run (without Compose)

1) Copy env:

```powershell
Copy-Item .env.example .env
```

2) Start Postgres + RabbitMQ (via Docker):

```powershell
docker compose up -d db rabbitmq
```

3) Apply migration manually (if DB volume already existed and init scripts were not re-run):

```powershell
Get-Content .\migrations\0001_init.sql | docker compose exec -T db psql -U postgres -d caspercloud
```

4) Run API:

```powershell
go run .\cmd\api
```

5) Run worker (second terminal):

```powershell
go run .\cmd\worker
```

## Build Verification

```powershell
go mod tidy
gofmt -w .\cmd .\internal
go build .\...
```

## API Summary

Auth:
- `POST /v1/auth/register`
- `POST /v1/auth/login`

Projects:
- `POST /v1/projects`
- `GET /v1/projects`

Images:
- `POST /v1/projects/{projectID}/images/`
- `GET /v1/projects/{projectID}/images/`
- `GET /v1/projects/{projectID}/images/{imageID}`
- `PUT /v1/projects/{projectID}/images/{imageID}`
- `DELETE /v1/projects/{projectID}/images/{imageID}`

Instances:
- `POST /v1/projects/{projectID}/instances/` (async create)
- `GET /v1/projects/{projectID}/instances/`
- `GET /v1/projects/{projectID}/instances/{instanceID}`
- `POST /v1/projects/{projectID}/instances/{instanceID}/start`
- `POST /v1/projects/{projectID}/instances/{instanceID}/stop`
- `POST /v1/projects/{projectID}/instances/{instanceID}/reboot`
- `DELETE /v1/projects/{projectID}/instances/{instanceID}`

OpenAPI spec: `api/openapi.yaml`

Sample requests: `docs/sample_requests.http`

## Tenant / Authorization Rules Implemented

- Instances belong to projects (`instances.project_id`).
- Users can only access resources in projects where they have membership.
- JWT is required for project/image/instance APIs.
- Instance create is asynchronous through RabbitMQ + worker.
- Instance state in DB is the source of truth and is updated after lifecycle operations.

## Notes on libvirt Adapter

`internal/libvirt/virsh_adapter.go` is a practical skeleton using `virsh`.
To productionize, add:
- domain XML generation
- cloud-init ISO generation and attachment
- storage/network provisioning
- richer state reconciliation and retries
