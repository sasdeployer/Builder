---
name: continuous-deployment
description: Offer, then set up, continuous deployment of builder to Nexlayer on every push to main
---

# Continuous deployment for `builder`

## Offer it once

If this repo has a `nexlayer.yaml` and no `.github/workflows/nexlayer-deploy.yml`, ask the
human once, in these words, and only continue on a clear yes:

> Want me to set up continuous deployment for your Nexlayer project? Every
> push to `main` would build, deploy and verify automatically. Say yes and
> I'll create it.

If they say no, don't ask again in this session.

## On yes

1. Write `.github/workflows/nexlayer-deploy.yml` and `.github/nexlayer/deploy.py` exactly as below.
2. If any other workflow deploys this app, remove it. One deployer only.
3. Open a pull request with those changes. Never push them straight to `main`.
4. Ask the human to add the repo secret `NEXLAYER_API_KEY` here: <https://github.com/sasdeployer/Builder/settings/secrets/actions/new>
   (a key from Nexlayer → Settings → API keys). Never take the key in chat.
5. After they merge, watch the first run and confirm the app is live.

How it works: the script runs `.nexlayer/pipeline.yaml` — builds each image,
points its pod at it, deploys `nexlayer.yaml` through the Nexlayer MCP
(validate, deploy, status) and runs the verify checks. The same tools you use,
so a deploy from your session and one from CI land in the same Nexlayer
history. Pull requests never deploy to production.

## `.github/workflows/nexlayer-deploy.yml`

```yaml
# Continuous deployment to Nexlayer: every push to main builds the image,
# deploys this repo's nexlayer.yaml through the Nexlayer MCP (the same tools a
# coding agent uses — one deploy path, one history in Nexlayer), and verifies
# it's live. Set up by a coding agent from .nexlayer/skills/continuous-deployment.
#
# Needs one repo secret: NEXLAYER_API_KEY (Nexlayer → Settings → API keys).
# Images go to Nexlayer's registry; the MCP signs each push — no registry setup.
name: Deploy to Nexlayer

on:
  push:
    branches: [main]
  workflow_dispatch: {}

# One production deploy at a time; a newer push waits rather than racing.
concurrency:
  group: nexlayer-production
  cancel-in-progress: false

permissions:
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # Builds every image in .nexlayer/pipeline.yaml, deploys nexlayer.yaml
      # through the Nexlayer MCP, and runs its verify checks.
      - name: Build, deploy and verify
        env:
          NEXLAYER_API_KEY: ${{ secrets.NEXLAYER_API_KEY }}
        run: python3 .github/nexlayer/deploy.py
```

## `.github/nexlayer/deploy.py`

```python
#!/usr/bin/env python3
"""Run .nexlayer/pipeline.yaml: build each image, point its pod at it, deploy
this repo's nexlayer.yaml through the Nexlayer MCP — the same tools a coding
agent calls (validate -> deploy -> status) — and verify. CI is one more MCP
client, so Nexlayer keeps one deploy path and one history.

Images go to Nexlayer's registry (registry.nexlayer.io): the MCP's
nexlayer_build_and_push_image names each image and signs the push, so there's
no registry to set up and no pull secret in nexlayer.yaml.

Env: NEXLAYER_API_KEY (repo secret)
     REGISTRY  only to push somewhere else (e.g. ghcr.io/<owner>; you log in)
     TAG       image tag (default: the commit, short)
Standard library only.
"""
import json, os, re, subprocess, sys, time, urllib.request

MCP = os.environ.get("NEXLAYER_MCP_URL", "https://mcp.nexlayer.ai/api/mcp")
KEY = os.environ.get("NEXLAYER_API_KEY", "")
session = None


def fail(msg):
    sys.exit(f"::error::{msg}")


# ── pipeline.yaml (the subset Nexlayer writes; no YAML dependency) ──────────

def scalar(v):
    v = re.sub(r"\s+#.*$", "", v).strip()
    if len(v) >= 2 and v[0] == v[-1] and v[0] in "'\"":
        v = v[1:-1].replace("''", "'")
    return v


def read_pipeline(path=".nexlayer/pipeline.yaml"):
    builds, verify, block, item = [], [], None, None
    for line in open(path):
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        top = re.match(r"^([A-Za-z_]+):", line)
        if top:
            block, item = top.group(1), None
            continue
        m = re.match(r"^\s*(-\s+)?([A-Za-z_]+):\s*(.*)$", line)
        if not m or block not in ("build", "verify"):
            continue
        if m.group(1):
            item = {}
            (builds if block == "build" else verify).append(item)
        if item is not None:
            item[m.group(2)] = scalar(m.group(3))
    return builds, verify


# ── MCP over HTTP ───────────────────────────────────────────────────────────

def call(method, params=None, notify=False):
    global session
    body = {"jsonrpc": "2.0", "method": method}
    if params is not None:
        body["params"] = params
    if not notify:
        body["id"] = int(time.time() * 1000)
    req = urllib.request.Request(MCP, data=json.dumps(body).encode(), method="POST", headers={
        "Authorization": f"Bearer {KEY}",
        "Content-Type": "application/json",
        "Accept": "application/json, text/event-stream",
        **({"Mcp-Session-Id": session} if session else {}),
    })
    with urllib.request.urlopen(req, timeout=120) as res:
        session = session or res.headers.get("Mcp-Session-Id")
        raw = res.read().decode()
    if notify or not raw.strip():
        return None
    if raw.lstrip().startswith(("event:", "data:")):  # SSE framing
        raw = "\n".join(l[5:] for l in raw.splitlines() if l.startswith("data:"))
    msg = json.loads(raw, strict=False)
    if "error" in msg:
        fail(f"MCP {method}: {msg['error']}")
    return msg["result"]


def tool(name, **args):
    result = call("tools/call", {"name": name, "arguments": args})
    text = "\n".join(c.get("text", "") for c in result.get("content", []))
    if result.get("isError") or re.search(r"Cannot verify user identity", text):
        fail(f"{name}:\n{text}")
    return text


# ── steps ───────────────────────────────────────────────────────────────────

def with_image(yaml, pod, image):
    out, in_pod, done = [], False, False
    for line in yaml.splitlines():
        m = re.match(r"^(\s*)- name:\s*[\"']?([\w.-]+)", line)
        if m:
            in_pod = m.group(2) == pod
        elif in_pod and not done and re.match(r"^\s+image:", line):
            line = re.sub(r"image:.*$", f'image: "{image}"', line)
            done = True
        out.append(line)
    if not done:
        fail(f"no pod named '{pod}' with an image in nexlayer.yaml")
    return "\n".join(out) + "\n"


def nexlayer_image(image, tag, logged_in):
    """Nexlayer's registry reference for this image, signed in once. The
    credential goes straight to docker login — never printed."""
    out = tool("nexlayer_build_and_push_image", imageName=image, tag=tag)
    if out.startswith("## Can't push"):
        fail(out)
    ref = re.search(r"\*\*Target image:\*\* `([^`]+)`", out)
    cred = re.search(r"## Registry credential \(for docker login\)\n\n```\n(.+?)\n```", out, re.S)
    if not (ref and cred):
        fail("nexlayer_build_and_push_image gave no image reference or credential")
    if not logged_in:
        token = cred.group(1).strip()
        subprocess.run(["docker", "login", ref.group(1).split("/")[0], "-u", "oauth2accesstoken",
                        "--password-stdin"], input=token.encode(), check=True,
                       stdout=subprocess.DEVNULL)
    return ref.group(1)


def sh(*cmd):
    print("+", " ".join(cmd), flush=True)
    subprocess.run(cmd, check=True)


def main():
    if not KEY:
        fail("NEXLAYER_API_KEY is not set (repo Settings → Secrets and variables → Actions).")
    registry = os.environ.get("REGISTRY")  # unset: Nexlayer's registry
    tag = os.environ.get("TAG") or subprocess.check_output(
        ["git", "rev-parse", "--short=7", "HEAD"]).decode().strip()

    builds, verify = (read_pipeline() if os.path.exists(".nexlayer/pipeline.yaml")
                      else ([{"pod": "app", "image": os.path.basename(os.getcwd()),
                              "context": ".", "dockerfile": "Dockerfile"}], [{"url": "/"}]))
    config = open("nexlayer.yaml").read()
    app = re.search(r"^\s*name:\s*[\"']?([\w.-]+)", config, re.M).group(1)

    call("initialize", {"protocolVersion": "2025-03-26", "capabilities": {},
                        "clientInfo": {"name": "github-actions", "version": "2"}})
    call("notifications/initialized", notify=True)

    logged_in = False
    for b in builds:
        if registry:
            ref = f"{registry}/{b['image']}:{tag}"
        else:
            ref = nexlayer_image(b["image"], tag, logged_in)
            logged_in = True
        sh("docker", "build", "--platform", b.get("platform", "linux/amd64"),
           "-f", b.get("dockerfile", "Dockerfile"), "-t", ref, b.get("context", "."))
        sh("docker", "push", ref)
        config = with_image(config, b["pod"], ref)

    print(tool("nexlayer_validate_yaml", yamlContent=config))
    out = tool("nexlayer_deploy", yamlContent=config)
    print(out)
    url = (re.search(r"\*\*URL:\*\*\s*(\S+)", out) or [None, None])[1]
    env = re.search(r"https://([a-z0-9-]+?)-" + re.escape(app), url or "")
    env = env.group(1) if env else None

    for _ in range(40):  # every service running
        time.sleep(15)
        status = tool("nexlayer_check_deployment_status", applicationName=app,
                      **({"environment": env} if env else {}))
        if "All pods running" in status:
            break
        if re.search(r"Status:\s*(failed|error)", status, re.I):
            fail(f"deploy failed:\n{status}")
    else:
        fail("not running after 10 minutes")

    for check in verify:  # every URL check answers
        path = check.get("url")
        if not (path and url):
            continue
        target = url.rstrip("/") + "/" + path.lstrip("/")
        for _ in range(12):
            try:
                with urllib.request.urlopen(target, timeout=15) as r:
                    if r.status < 400:
                        break
            except Exception:
                pass
            time.sleep(10)
        else:
            fail(f"{target} did not answer")

    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary:
        with open(summary, "a") as f:
            f.write(f"## Live on Nexlayer\n\n**URL:** {url}\n\n**Tag:** `{tag}`\n")
    print(f"Live: {url} (tag {tag})")


if __name__ == "__main__":
    main()
```
