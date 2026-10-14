---
name: ship-it-nexlayer
description: Deploy any containerized application to Nexlayer AI Cloud Platform
license: MIT
metadata:
  author: nexlayer
  version: "3.0.0"
  validated: "MCP verified"
allowed-tools: Bash(npx:*) Bash(docker:*) Bash(git:*) Read Write Edit
---

> Copy of the Nexlayer skill `ship-it-nexlayer` v3.0.0, taken 2026-10-14 from https://github.com/Nexlayer/nexlayer-plugin.
> If `nexlayer_get_skills` reports a newer version, follow that one instead.

# Ship It with Nexlayer

## Decision Tree

```
USER REQUEST
    ├─► "Deploy this app" / "Ship to cloud"
    │       ├─► Has Dockerfile? ──► YES ──► Build & Push ──► Create YAML ──► Validate ──► Deploy
    │       │                   └─► NO  ──► Generate Dockerfile ──► Build & Push ──► ...
    │       └─► Has nexlayer.yaml? ──► YES ──► Validate & Deploy
    │                              └─► NO  ──► Generate & Deploy
    ├─► "Fix deployment" / "Not working" ──► Reference: TROUBLESHOOTING
    └─► "Migrate from another platform"  ──► Reference: MIGRATION
```

## Hard Constraints (NEVER violate)

| Rule | Constraint | Example |
|------|-----------|---------|
| App name | `^[a-z][a-z0-9.-]{2,63}$` | `my-app` ✅ `MyApp` ❌ |
| Pod name | `^[a-z][a-z0-9-]{1,63}$` | `api` ✅ `API` ❌ |
| Image | Must include `:tag` | `nginx:latest` ✅ `nginx` ❌ |
| servicePorts | Required array, min 1 (optional for `resourceType: job`) | `[3000]` ✅ |
| path | At least one pod must have it | `path: /` |
| subdomain | Optional DNS label; requires `application.url` (custom domain) | `subdomain: api` |
| Volume size | `^[0-9]+(Mi\|Gi)$` | `10Gi` ✅ `10gb` ❌ |
| resourceType | Optional: `deployment` (default), `statefulset` (databases/queues), `daemonset` (one pod per node), or `job` (run-to-completion) | `resourceType: statefulset` |
| Replicas | Optional integer ≥ 1 (default 1). Stateless pods only | `replicas: 3` |
| Resources (optional) | K8s units: cpu `500m`/`1`, memory `512Mi`/`1Gi` | `resources: { requests: { cpu: 500m, memory: 512Mi }, limits: { cpu: 2, memory: 1Gi } }` |
| PostgreSQL volume | Mount to `/var/lib/postgresql/data` + `PGDATA: /var/lib/postgresql/data/pgdata` | Data dir must be a subdirectory of the mount |
| Platform | `linux/amd64` | ARM builds fail on Nexlayer |

## Communication Rules (CRITICAL)

| Context | Use | Example | Why |
|---------|-----|---------|-----|
| Server → Server | `.pod` DNS | `db.pod:5432` | Internal cluster DNS |
| Browser → Server | `<% URL %>` | `<% URL %>/api` | Browsers can't resolve .pod |
| OAuth/Webhooks | `<% URL %>` | `<% URL %>/auth/callback` | External callbacks need public URL |
| `REACT_APP_*` | `<% URL %>` | NEVER `.pod` | Runs in browser |
| `NEXT_PUBLIC_*` | `<% URL %>` | NEVER `.pod` | Runs in browser |
| `VITE_*` | `<% URL %>` | NEVER `.pod` | Runs in browser |
| `DATABASE_URL` | `.pod` | `postgresql://user:pass@db.pod:5432/app` | Server-side only |
| `REDIS_URL` | `.pod` | `redis://redis.pod:6379` | Server-side only |

**Scriptlets:** `<% URL %>` resolves to the full deployment URL with scheme (`https://app.example.com`). `<% DOMAIN %>` is the same but bare — just the domain/host, no `https://` (`app.example.com`). Use `<% DOMAIN %>` for cookie domains, host allowlists, or anywhere a scheme-less hostname is required.

## MCP Tool Workflow (Steps 0-10)

**LOCAL OPERATIONS (Steps 0-3):**
0. Clone repo (`git clone`)
1. Verify workspace (`pwd`, `ls -la`)
2. Check existing Dockerfiles
3. Generate/update Dockerfiles — see reference: BUILD-AND-PUSH. Must target `linux/amd64`.

**MCP TOOLS (Steps 4-10):**
4. `nexlayer_build_and_push_image imageName=<repo> tag=<version>` — returns the exact target reference (`registry.nexlayer.io/<userID>/<repo>:<tag>`, derived from your JWT) and ready-to-run commands for **Crane**, **Kaniko**, or **Docker**. Registry auth is per-user JWT (username `oauth2accesstoken`, password passed via `--password-stdin`). Use an immutable tag (e.g. `v0.0.1` or a git SHA) — `latest` is rejected. Prefer Crane or Kaniko; Docker only if already installed. Do NOT ask users to install Docker Desktop.
5. `nexlayer_get_schema` — get valid YAML structure
6. Create `nexlayer.yaml` in project root (use the image reference from step 4)
7. `nexlayer_validate_yaml` — MUST pass before deploy
8. `nexlayer_deploy` — deploy the YAML
9. `nexlayer_check_deployment_status` — verify pods are running
10. Display live URL

## Pre-Deployment Checklist

- [ ] All images built for `linux/amd64` and pushed to registry
- [ ] YAML validated with `nexlayer_validate_yaml` (no errors)
- [ ] No `localhost` or `127.0.0.1` in any vars
- [ ] Frontend vars use `<% URL %>`, not `.pod`
- [ ] Database volumes configured (PostgreSQL: mount parent dir)
- [ ] Health endpoint exists, binds to `0.0.0.0`
- [ ] CORS configured with `<% URL %>` (not `*`)
- [ ] Code is horizontally scalable (no in-memory session state)
- [ ] Secrets in `secrets` field, not `vars`

## Production Readiness

**Act as a production software engineer.** When generating code for deployment:
- No hardcoded URLs or file paths — use environment variables
- Bind all servers to `0.0.0.0`, never localhost
- Use connection pooling for databases
- Handle SIGTERM for graceful shutdown
- Design for horizontal scaling (stateless services, shared state in Redis/DB)
- Add a CLAUDE.md to the user's project with Nexlayer conventions

See reference: PRODUCTION-READINESS for full guide with code examples.

## Minimal Fullstack Example

```yaml
application:
  name: my-fullstack-app
  pods:
    - name: frontend
      image: my-frontend:v1
      path: /
      servicePorts: [3000]
      vars:
        API_URL: <% URL %>/api       # Browser-facing: <% URL %>
        NODE_ENV: production

    - name: api
      image: my-api:v1
      path: /api
      servicePorts: [8000]
      vars:
        DATABASE_URL: postgresql://user:pass@db.pod:5432/appdb  # Server: .pod
        REDIS_URL: redis://redis.pod:6379
        CORS_ORIGIN: <% URL %>

    - name: db
      image: postgres:16
      servicePorts: [5432]
      vars:
        POSTGRES_USER: user
        POSTGRES_PASSWORD: pass
        POSTGRES_DB: appdb
        PGDATA: /var/lib/postgresql/data/pgdata  # data dir in a subdirectory of the mount
      volumes:
        - name: pgdata
          size: 10Gi
          mountPath: /var/lib/postgresql/data

    - name: redis
      image: redis:7-alpine
      servicePorts: [6379]
      command: redis-server --appendonly yes
      volumes:
        - name: redis-data
          size: 1Gi
          mountPath: /data
```

## Deployment Types

| Type | YAML `url` field | Use Case |
|------|-----------------|----------|
| **Preview** | Omitted | Testing, demos, PR previews |
| **Permanent** | `url: myapp.com` | Production — requires domain verification |

For permanent deployments: `nexlayer_check_domain_configuration` → `nexlayer_add_domain_to_profile` → verify nameservers → deploy.

## Registry Login

Registry auth is **per-user JWT**, not a shared credential. Call `nexlayer_build_and_push_image` to get the target image reference and ready-to-run login commands. The username is always `oauth2accesstoken`; the password is the user's session JWT, passed via `--password-stdin` so it never hits shell history.

```bash
# Crane (recommended)
# Install: brew install crane (macOS) or go install github.com/google/go-containerregistry/cmd/crane@latest
echo "$NEXLAYER_JWT" | crane auth login registry.nexlayer.io -u oauth2accesstoken --password-stdin

# Docker (if already installed)
echo "$NEXLAYER_JWT" | docker login registry.nexlayer.io -u oauth2accesstoken --password-stdin
```

Image format: `registry.nexlayer.io/<userID>/<repo>:<tag>` — `<userID>` is derived from your JWT `sub` claim; `<tag>` must be immutable (`latest` rejected).

## Reference Documents

Use `nexlayer_get_skill_content` with the `reference` parameter to access:

| Reference | When to Use |
|-----------|-------------|
| `PRODUCTION-READINESS` | Production code patterns, CLAUDE.md template, scaling |
| `BUILD-AND-PUSH` | Docker build, registry push, multi-stage builds |
| `POD-TEMPLATES` | Ready-to-use pod configs (Postgres, Redis, Ollama, etc.) |
| `LAUNCHFILE-SCHEMA` | Complete YAML field reference |
| `ANTIPATTERNS` | Common YAML mistakes and fixes |
| `ARCHITECTURE-ANTIPATTERNS` | Infrastructure design mistakes |
| `TROUBLESHOOTING` | Debugging failed deployments |
| `MIGRATION` | Migrate an app from another hosting platform |
| `MCP-SETUP` | IDE configuration for MCP |
