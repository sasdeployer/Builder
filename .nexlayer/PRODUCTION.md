# Nexlayer — how `builder` ships

You were asked to work on or deploy this app. The state below is already
resolved — do not re-derive it from the code. The procedure is not here;
ask Nexlayer for it (see "How to deploy").

## The app

| | |
| --- | --- |
| Name | `builder` |
| Repo | `https://github.com/sasdeployer/Builder` on `main` |
| Planned | 2026-10-14T19:33:33.479Z |
| Registered with Nexlayer | yes |

`.nexlayer/plan.lock` pins the commit this plan was written against. If HEAD
has moved and you changed how the app starts, runs, or what it needs,
re-check before deploying.

## The production plan

Written by the Nexlayer agent from this repo. Every decision cites the files it
rests on; if the code has changed since, re-check those files first.

Builder is a Go CLI that clones a repo, detects its language, compiles or packages it, and emits a zip artifact with build metadata. It runs to completion as a batch job with no web server or persistent daemon. The one thing that matters most is that the Dockerfile lacks a CMD to actually execute the built binary, so the job currently exits without doing any work.

- **services: Single pod running as a Kubernetes job that executes a build and exits; no companion pods** — Builder is a CLI tool invoked with `builder init [repo url]` or `builder` from a workspace — it performs a one-shot build and terminates, with no listening port or long-running process (`main.go`, `cmd/builder.go`, `cmd/init.go`)
- **storage: A 5 Gi volume mounted at /data for workspace, logs, artifact, and hidden directories that must survive restarts** — The directory/ package creates hidden, logs, workspace, and artifact dirs on disk; without persistence the built artifacts and metadata are lost after the job pod is rescheduled (`directory/workspace.go`, `directory/logs.go`, `artifact/artifactDir.go`, `directory/hidden.go`)
- **build: Use golang:1.16-alpine matching go.mod's go 1.16 directive, add ENV GOTOOLCHAIN=auto, tini, ca-certificates, and a CMD that runs /go/src/builder/builder** — The existing Dockerfile builds the binary but has no CMD or ENTRYPOINT so the container exits immediately without executing it; tini ensures proper signal handling and ca-certificates enables git clone over HTTPS (`Dockerfile`, `go.mod`)
- **background_work: Set resourceType to job (application.version 2.0) — Builder is a run-to-completion workload, not a deployment** — Builder's main() dispatches a single command (init, config, or build) and returns; it never enters a loop or serves requests (`main.go`, `cmd/builder.go`)
- **networking: No servicePorts and no path — the job pod exposes no network endpoint** — Builder only uses outbound git clone; it never listens on a port (`utils/cloneRepo.go`, `main.go`)

### Fix before production

- **Blocker** — Add a CMD or ENTRYPOINT to execute the built binary: The Dockerfile compiles /go/src/builder/builder but has no CMD — the container starts, does nothing, and exits immediately, so every job run completes with zero output (`Dockerfile`)
- Remove or gate the GUI mode (lorca) behind a build tag or runtime check: main.go unconditionally imports the gui package which calls lorca (requires a native Chrome binary); running `builder gui` inside the container will panic at runtime (`main.go`)
- Install language toolchains (dotnet-sdk, openjdk, node, python3, ruby) in the Docker image: The compile/ package shells out to `dotnet publish`, `javac`, `npm`, `python`, `ruby` — without them every non-Go build silently fails inside the container (`Dockerfile`)

### Verify after the deploy

1. `builder init https://github.com/example/go-hello` completes and produces a workspace directory with a builder.yaml and an artifact zip
2. The artifact directory contains both .json and .yaml metadata files alongside the zip
3. The job pod exits with exit code 0 and logs show 'Build Complete 🔨'
4. The volume at /data retains the artifact zip after the pod completes (check volume contents before reschedule)

### Ask the human

- Which target languages must be buildable in production? This determines which toolchains to install in the image (Go only vs. all six supported languages).
- Should this run as an on-demand Job (triggered externally) or a CronJob with a schedule (e.g., nightly rebuilds)?
- Does the GUI mode need to remain in the binary at all, or can it be excluded from the production build via a separate entrypoint?

## Drafts in this plan

This repo had no deploy config, so this plan adds drafts where files were
missing (never over an existing file):

- `nexlayer.yaml` — what runs, written by the Nexlayer agent (see "The production plan"). It passes the Nexlayer validator.

Build them once, fix what fails, then deploy with `.nexlayer/pipeline.yaml`.
After the first successful deploy, these files are the source of truth.

## Can this deploy right now?

**Not yet — 1 blocker under "Fix before production" (yours).** Full list in `.nexlayer/todo.md`.

Fix those first. Deploying with them open ships a known problem.

## How to deploy

Call `nexlayer_get_deployment_workflow` first. It returns the current
procedure — building and pushing the image included — and it is kept up to
date in a way this file is not. Do not infer the steps from here, and do
not skip it because the app looks simple.

If Nexlayer tools are not available to you, the human runs
`npx @nexlayer/mcp-install` once.

## Secrets

This app needs no secrets.

## What was inferred rather than read

Nothing. Every claim in this plan was read from the repo.

## What "it worked" means

After every deploy, run `nexlayer_verify_deploy`. A deploy is accepted before it is
known to work; only "✓ Working" means it worked. It runs the `verify` list in
`.nexlayer/pipeline.yaml`, and when the app is broken it says what broke and why.

## Stop and ask the human

- A required key is missing (send the link above — never take the value).
- Something would become publicly reachable that is internal in this plan.
- Anything that deletes data or tears down a running deployment.

Everything else is yours to do. When something breaks, start at
`.nexlayer/TROUBLESHOOTING.md`.
