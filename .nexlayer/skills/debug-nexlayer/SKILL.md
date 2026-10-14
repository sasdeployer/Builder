---
name: debug-nexlayer
description: Debug and troubleshoot live Nexlayer deployments — exec into pods, edit files, query databases, restart services
license: MIT
metadata:
  author: nexlayer
  version: "1.1.0"
  validated: "MCP verified"
allowed-tools: nexlayer_debug_*
---

> Copy of the Nexlayer skill `debug-nexlayer` v1.1.0, taken 2026-10-14 from https://github.com/Nexlayer/nexlayer-plugin.
> If `nexlayer_get_skills` reports a newer version, follow that one instead.

# Debug Mode — Nexlayer Deployments

## CRITICAL RULES — Read Before Doing Anything

1. **You MUST have the applicationName before calling deploy_proxy.** The applicationName is the name field from the user's `nexlayer.yaml`. If you don't know it, ASK the user. Do NOT deploy the proxy without it — unscoped sessions dump all pods in the namespace and waste context.

2. **Deploy the proxy ONCE per debug session.** After deploy_proxy succeeds, the proxy stays running. Do NOT call destroy_proxy and redeploy unless switching to a different namespace. The proxy auto-scales to 0 after 10 minutes of inactivity and wakes up on next tool call.

3. **Only call the tools you need for the current problem.** Do NOT enumerate or "test" all available tools. Follow the decision tree below to pick the 2-4 tools that match the user's symptom.

4. **Always start with namespace_info after deploying.** This gives you the pod names, statuses, and services you need for subsequent tool calls. Read its output carefully before calling anything else.

5. **After making changes (file_edit, file_copy_to, env changes), you MUST restart the affected pod.** Use `pod_restart` for a single pod or `pod_restart_deployment` for all replicas. Changes are NOT picked up until the pod restarts.

## Getting the applicationName

The applicationName comes from the user's `nexlayer.yaml` file. It is the top-level `name` field. Examples:

- If `nexlayer.yaml` has `name: my-saas-app`, the applicationName is `my-saas-app`
- If `nexlayer.yaml` has `name: api-backend`, the applicationName is `api-backend`

**How to get it:**
1. **Best:** Ask the user directly: "What is the applicationName from your nexlayer.yaml?"
2. **If user gives a domain:** Call `nexlayer_get_deployments` to list deployments, find the one matching the domain, and extract the applicationName from the response.
3. **If user gives a URL:** The domain is the noun-noun prefix (e.g., `happy-tiger` from `happy-tiger.nexlayer.app`). Then use option 2.

**Never guess the applicationName.** If you deploy the proxy without it, all debug operations return unfiltered data from every pod in the namespace, which wastes your context window and risks operating on the wrong pods.

## Decision Tree — What To Call and When

Follow this tree based on what the user is asking. Call ONLY the tools in the path that matches.

```
USER REQUEST
    |
    +-- "Debug my app" / "Something is broken" / general debugging
    |       1. Ask user for applicationName + domain (or get from nexlayer_get_deployments)
    |       2. nexlayer_debug_deploy_proxy(domain, applicationName)
    |       3. nexlayer_debug_namespace_info(domain) — read pod names + statuses
    |       4. Investigate based on what namespace_info shows:
    |           - Pods not Running? → pod_describe to check events
    |           - CrashLoopBackOff? → pod_describe → shell_open → check logs
    |           - All pods Running but broken? → proxy_http to test endpoints
    |
    +-- "Check logs" / "Why is it crashing?"
    |       1. Try nexlayer_get_deployment_logs FIRST (no proxy needed)
    |       2. Only if logs aren't enough → deploy proxy → shell_open → check files/config
    |
    +-- "Fix a config file" / "Change environment variable"
    |       1. Deploy proxy (if not already running)
    |       2. nexlayer_debug_file_copy_from — read the current file
    |       3. nexlayer_debug_file_edit — make the change
    |       4. nexlayer_debug_pod_restart — restart so the pod picks up changes
    |
    +-- "Database issue" / "Check the data"
    |       1. Deploy proxy (if not already running)
    |       2. nexlayer_debug_db_query — run SQL
    |
    +-- "Service not responding" / "Network issue"
    |       1. Deploy proxy (if not already running)
    |       2. nexlayer_debug_proxy_http — test the endpoint from inside the namespace
    |       3. If that fails → nexlayer_debug_namespace_dns — check DNS resolution
    |       4. If DNS ok but service down → nexlayer_debug_pod_describe — check pod health
    |
    +-- "Done debugging" / user is finished
            1. nexlayer_debug_destroy_proxy — clean up
            (Or just leave it — auto-scales to 0 in 10 minutes)
```

## Proxy Lifecycle Management

| Action | When |
|--------|------|
| `deploy_proxy` | Once at the start of a debug session. Always pass `applicationName`. |
| Re-use existing proxy | For ALL subsequent tool calls in the same namespace. Do NOT redeploy. |
| `destroy_proxy` | Only when the user says they're done, or when switching to a different namespace. |
| Proxy went idle | If the proxy scaled to 0 (10 min idle), just call any debug tool — it auto-restarts. No need to redeploy. |
| Switch application | If debugging a different app in the SAME namespace, call `deploy_proxy` again with the new `applicationName`. This re-scopes but doesn't recreate the pod. |

**DO NOT** destroy and redeploy the proxy between tool calls. Deploying a proxy takes 30-60 seconds. Reuse it.

## Workflow Details

### Step 1: Deploy the Proxy (ALWAYS pass applicationName)

```
nexlayer_debug_deploy_proxy
  domain: "happy-tiger"
  applicationName: "my-app"    # REQUIRED — from nexlayer.yaml
```

The tool blocks until the proxy is healthy (up to 60s). When it returns success, all other tools work immediately.

### Step 2: Get Namespace Info (ALWAYS do this next)

```
nexlayer_debug_namespace_info
  domain: "happy-tiger"
```

Returns pods, services, and configmaps **filtered to your applicationName**. This gives you the exact pod names you'll need for shell_open, file operations, pod_restart, etc.

Read the output carefully. Note which pods are Running, which are failing, and what services exist.

### Step 3: Investigate Based on Symptoms

**Interactive Shell (SSH-like) — for deep investigation:**

```
nexlayer_debug_shell_open
  domain: "happy-tiger"
  pod: "api-my-app-6dbf886659-fqqx7"    # pod name from namespace_info
```

Returns a `shellSessionId`. Use it for subsequent commands:

```
nexlayer_debug_shell_send
  domain: "happy-tiger"
  shellSessionId: "abc123"     # from shell_open response — NOT sessionId
  command: "cat /app/config.yaml"
```

Shell state persists between calls — `cd`, environment variables, aliases all maintained.

Close when done investigating:

```
nexlayer_debug_shell_close
  domain: "happy-tiger"
  shellSessionId: "abc123"
```

**File Operations (SCP-like) — for reading/editing config:**

Read a file:
```
nexlayer_debug_file_copy_from
  domain: "happy-tiger"
  pod: "api-my-app-..."
  path: "/app/config.yaml"
```

Edit in-place (find-and-replace):
```
nexlayer_debug_file_edit
  domain: "happy-tiger"
  pod: "api-my-app-..."
  path: "/app/config.yaml"
  search: "old_value"
  replace: "new_value"
```

Write/overwrite a file:
```
nexlayer_debug_file_copy_to
  domain: "happy-tiger"
  pod: "api-my-app-..."
  path: "/app/config.yaml"
  content: "... new content ..."
```

List directory:
```
nexlayer_debug_file_list
  domain: "happy-tiger"
  pod: "api-my-app-..."
  path: "/app"
```

**IMPORTANT: After editing files, restart the pod to pick up changes:**
```
nexlayer_debug_pod_restart
  domain: "happy-tiger"
  pod: "api-my-app-..."
```

**Database Queries:**

```
nexlayer_debug_db_query
  domain: "happy-tiger"
  connectionString: "postgresql://user:pass@database-my-app.pod:5432/appdb"
  query: "SELECT * FROM users LIMIT 10"
```

For MySQL: `connectionString: "mysql://user:pass@database-my-app.pod:3306/appdb"`

The proxy has `psql` and `mysql` CLI tools. Connection strings use `.pod` DNS within the namespace.

**Pod Lifecycle:**

Restart a specific pod:
```
nexlayer_debug_pod_restart
  domain: "happy-tiger"
  pod: "api-my-app-..."
```

Rolling restart of entire deployment:
```
nexlayer_debug_pod_restart_deployment
  domain: "happy-tiger"
  deployment: "api-my-app"
```

Scale a deployment:
```
nexlayer_debug_pod_scale
  domain: "happy-tiger"
  deployment: "api-my-app"
  replicas: 0    # scale to 0 then back to 1 for hard restart
```

Full pod description (spec, status, events):
```
nexlayer_debug_pod_describe
  domain: "happy-tiger"
  pod: "api-my-app-..."
```

**Network Testing:**

HTTP request from inside the namespace:
```
nexlayer_debug_proxy_http
  domain: "happy-tiger"
  method: "GET"
  url: "http://api-my-app:8000/health"
```

Run arbitrary command from proxy:
```
nexlayer_debug_proxy_exec
  domain: "happy-tiger"
  command: "curl -s http://api-my-app:8000/health"
```

DNS resolution:
```
nexlayer_debug_namespace_dns
  domain: "happy-tiger"
  hostname: "api-my-app"
```

### Cleanup

The proxy auto-scales to 0 after 10 minutes of inactivity. Manual removal:

```
nexlayer_debug_destroy_proxy
  domain: "happy-tiger"
```

## Cross-Context Safety

When you call `deploy_proxy` with an `applicationName`, all operations are scoped:

- `namespace_info` filters pods to only show your app's pods
- Any tool targeting a pod outside your app's scope returns a **WARNING**
- To switch applications, call `deploy_proxy` again with the new `applicationName`

This prevents accidentally restarting the wrong pod or editing files in the wrong container.

## Large Output Handling

Debug tools can return large data. The server automatically manages this:

1. **Small responses (< 4KB)**: Returned inline.
2. **Large responses (>= 4KB)**: Returns a **compact summary** + a **reference ID** (e.g. `ref-a1b2c3d4`). Full data is cached server-side for 30 minutes.

To retrieve full data:

```
nexlayer_debug_fetch_result
  ref: "ref-a1b2c3d4"          # reference ID from summary
  offset: 0                     # optional: line offset (default 0)
  limit: 200                    # optional: max lines (default 200)
  search: "error"               # optional: only lines containing this text
```

**Key pattern**: Use `search` to grep through cached results without loading everything. For example, after `pod_describe` returns a summary showing CrashLoopBackOff, use `fetch_result(ref: "ref-xxx", search: "Error")` to find just the error-related lines.

**Additional tool parameters:**
- `file_copy_from`: `maxLines` (int) and `tail` (bool) to read only first/last N lines
- `db_query`: `limit` (int, default 100) to cap rows. Auto-injects SQL LIMIT if not present.

## Troubleshooting Patterns

| Symptom | Debug Steps |
|---------|-------------|
| Pod crash-looping | `namespace_info` → `pod_describe` → check events → `shell_open` → check logs/config |
| Service not reachable | `proxy_http` to test → `namespace_dns` to check DNS → `proxy_exec` with `netstat` |
| Wrong config deployed | `file_copy_from` to read → `file_edit` to fix → `pod_restart` to reload |
| Database migration failed | `db_query` to check schema → `shell_open` to run migration manually |
| Out of memory | `pod_describe` for resource limits → `proxy_exec` with `ps aux` in pod |
| Image pull errors | `pod_describe` → check events for image pull status |

## Available Tools Quick Reference

| Tool | Purpose | Requires Pod? |
|------|---------|---------------|
| `deploy_proxy` | Start debug session (ALWAYS pass applicationName) | No |
| `destroy_proxy` | End debug session (only when done) | No |
| `fetch_result` | Retrieve cached large output by ref ID | No |
| `namespace_info` | List pods/services (filtered to app) | No |
| `namespace_dns` | DNS resolution test | No |
| `proxy_exec` | Run command from proxy | No |
| `proxy_http` | HTTP request from namespace | No |
| `shell_open` | SSH-like session to pod | Yes |
| `shell_send` | Send command to session (uses shellSessionId) | No |
| `shell_close` | End shell session (uses shellSessionId) | No |
| `shell_list` | List active sessions | No |
| `file_copy_from` | Read file from pod | Yes |
| `file_copy_to` | Write file to pod | Yes |
| `file_edit` | Edit file in-place | Yes |
| `file_list` | List directory | Yes |
| `db_query` | SQL query | No |
| `pod_restart` | Restart pod | Yes |
| `pod_restart_deployment` | Rolling restart | No (uses deployment name) |
| `pod_scale` | Scale deployment | No (uses deployment name) |
| `pod_describe` | Pod details + events | Yes |
