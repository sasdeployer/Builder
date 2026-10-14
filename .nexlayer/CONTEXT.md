# builder — production context

Written by the Nexlayer agent so any coding agent that opens this repo
starts with the same picture. Read this before proposing infrastructure
changes.

- **Repo** `https://github.com/sasdeployer/Builder` on `main`
- **Analyzed** 2026-10-14T19:33:33.479Z

## Stack

| Component | Version | How we know |
| --- | --- | --- |
| Go | 1.16 | read from `go.mod`, `Dockerfile` |
| promptui | 0.8.0 | read from `go.mod` |
| lorca | 0.1.10 | read from `go.mod` |
| zap | 1.24.0 | read from `go.mod` |
| yaml.v3 | 3.0.1 | read from `go.mod` |
| yacspin | 0.13.12 | read from `go.mod` |

## How Nexlayer will run it

| Service | Reachable | Image | How we know |
| --- | --- | --- | --- |
| `builder` | internal only | `registry.nexlayer.io/YOUR_USER_ID/builder-builder:planned` | read from `.nexlayer/drafts/nexlayer.yaml` |

Reachability is inferred from service names and roles, not stated by the
analysis. Check it before relying on it — exposing something that should
be internal is not recoverable by editing this file afterwards.

Networking, HTTPS, and service discovery are handled.

## Secrets

This app needs no secrets to run.

## What the human told us

**Stage.** This is a side project.

Said by a person, not derived from the code. Where this contradicts what
the repo looks like, the person is right about intent and the repo is
right about what exists today.

## Notes from the analysis

- Builder is a CLI/batch tool, not a long-running web service; deploy as a job-type pod that executes builds and exits.
- The container mounts a volume at /data (or similar) for persistent artifact and metadata storage across invocations.
- No database pod is required; metadata is stored as flat YAML/JSON files within the artifact directory.
- The gui/ directory uses lorca (native Chrome webview) and is incompatible with headless container environments; only the CLI build pipeline is deployable on-cluster.
- To build multi-language targets inside the container, install language toolchains (go, dotnet, javac, node, python3, ruby) in the image or mount them as sidecar volumes.

## Talking to Nexlayer

Nexlayer is reachable over MCP. Call `nexlayer_get_deployment_workflow`
before deploying — it is the current procedure, and it changes more often
than this file does.
