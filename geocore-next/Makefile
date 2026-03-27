.PHONY: help dev dev-deps build push deploy rollback logs status clean

  REGISTRY   := ghcr.io/hossam-create
  TAG        := $(shell git rev-parse --short HEAD)
  NAMESPACE  := geocore

  help:
      @echo ""
      @echo "  GeoCore — Build & Deploy Commands"
      @echo "  ────────────────────────────────────────────────────────────"
      @echo "  Local Development:"
      @echo "    make dev           Start full local stack (Docker Compose)"
      @echo "    make dev-deps      Start only Postgres + Redis"
      @echo "    make run-backend   Run Go API locally"
      @echo "    make run-frontend  Run Next.js locally"
      @echo ""
      @echo "  Docker:"
      @echo "    make build         Build all Docker images"
      @echo "    make push          Push images to ghcr.io"
      @echo ""
      @echo "  Kubernetes:"
      @echo "    make deploy        Apply all K8s manifests"
      @echo "    make rollback      Rollback API to previous deployment"
      @echo "    make logs          Tail API pod logs"
      @echo "    make status        Show pod/service status"
      @echo "    make scale-api     Scale API to N replicas (N=3)"
      @echo ""

  # ── Local Development ─────────────────────────────────────────────────────────
  dev:
      docker compose -f infra/docker/docker-compose.yml up -d
      @echo "✅  Stack started:"
      @echo "    Web:    http://localhost:3000"
      @echo "    Admin:  http://localhost:3001"
      @echo "    API:    http://localhost:8080"
      @echo "    DB:     localhost:5432"

  dev-stop:
      docker compose -f infra/docker/docker-compose.yml down

  dev-deps:
      docker compose -f infra/docker/docker-compose.yml up -d postgres redis

  run-backend:
      cd backend && go run ./cmd/api

  run-frontend:
      cd frontend && npm run dev

  install:
      cd frontend && npm install
      cd admin && npm install

  # ── Docker Build & Push ───────────────────────────────────────────────────────
  build:
      docker build -f infra/docker/Dockerfile.api      -t $(REGISTRY)/geocore-api:$(TAG)      -t $(REGISTRY)/geocore-api:latest      .
      docker build -f infra/docker/Dockerfile.frontend  -t $(REGISTRY)/geocore-frontend:$(TAG)  -t $(REGISTRY)/geocore-frontend:latest  .
      docker build -f infra/docker/Dockerfile.admin     -t $(REGISTRY)/geocore-admin:$(TAG)     -t $(REGISTRY)/geocore-admin:latest     .
      @echo "✅  Built: api, frontend, admin @ $(TAG)"

  push: build
      docker push $(REGISTRY)/geocore-api:$(TAG)
      docker push $(REGISTRY)/geocore-api:latest
      docker push $(REGISTRY)/geocore-frontend:$(TAG)
      docker push $(REGISTRY)/geocore-frontend:latest
      docker push $(REGISTRY)/geocore-admin:$(TAG)
      docker push $(REGISTRY)/geocore-admin:latest
      @echo "✅  Pushed all images to ghcr.io"

  # ── Kubernetes ────────────────────────────────────────────────────────────────
  deploy:
      @echo "🚀  Deploying GeoCore to Kubernetes..."
      kubectl apply -k infra/k8s/
      kubectl rollout status deployment/geocore-api       -n $(NAMESPACE) --timeout=120s
      kubectl rollout status deployment/geocore-frontend  -n $(NAMESPACE) --timeout=120s
      kubectl rollout status deployment/geocore-admin     -n $(NAMESPACE) --timeout=60s
      @echo "✅  Deployment complete"

  rollback:
      kubectl rollout undo deployment/geocore-api -n $(NAMESPACE)
      @echo "⏪  Rolled back geocore-api"

  logs:
      kubectl logs -n $(NAMESPACE) -l app=geocore-api --tail=100 -f

  status:
      @echo "── Pods ──────────────────────────────────────"
      kubectl get pods -n $(NAMESPACE) -o wide
      @echo ""
      @echo "── Services ──────────────────────────────────"
      kubectl get svc -n $(NAMESPACE)
      @echo ""
      @echo "── Ingress ───────────────────────────────────"
      kubectl get ingress -n $(NAMESPACE)
      @echo ""
      @echo "── HPA ───────────────────────────────────────"
      kubectl get hpa -n $(NAMESPACE)

  scale-api:
      kubectl scale deployment geocore-api -n $(NAMESPACE) --replicas=$(N)
      @echo "✅  Scaled geocore-api to $(N) replicas"

  # ── Utilities ─────────────────────────────────────────────────────────────────
  db-shell:
      kubectl exec -it -n $(NAMESPACE) $$(kubectl get pod -n $(NAMESPACE) -l app=postgres -o jsonpath='{.items[0].metadata.name}') -- psql -U geocore geocore

  redis-shell:
      kubectl exec -it -n $(NAMESPACE) $$(kubectl get pod -n $(NAMESPACE) -l app=redis -o jsonpath='{.items[0].metadata.name}') -- redis-cli

  clean:
      docker compose -f infra/docker/docker-compose.yml down -v
      @echo "🗑️   Removed containers and volumes"
  