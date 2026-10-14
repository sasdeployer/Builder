# builder — before this ships

Two lists, split by who can actually close the item.

## Needs code — the coding agent

- [ ] **Blocker:** Add a CMD or ENTRYPOINT to execute the built binary (`Dockerfile`)
      _The Dockerfile compiles /go/src/builder/builder but has no CMD — the container starts, does nothing, and exits immediately, so every job run completes with zero output_
- [ ] Remove or gate the GUI mode (lorca) behind a build tag or runtime check (`main.go`)
      _main.go unconditionally imports the gui package which calls lorca (requires a native Chrome binary); running `builder gui` inside the container will panic at runtime_
- [ ] Install language toolchains (dotnet-sdk, openjdk, node, python3, ruby) in the Docker image (`Dockerfile`)
      _The compile/ package shells out to `dotnet publish`, `javac`, `npm`, `python`, `ruby` — without them every non-Go build silently fails inside the container_
- [ ] Build the drafted image(s) once and fix what fails; check `nexlayer.yaml`.

## Needs the human

- [ ] Decide: Which target languages must be buildable in production? This determines which toolchains to install in the image (Go only vs. all six supported languages).
- [ ] Decide: Should this run as an on-demand Job (triggered externally) or a CronJob with a schedule (e.g., nightly rebuilds)?
- [ ] Decide: Does the GUI mode need to remain in the binary at all, or can it be excluded from the production build via a separate entrypoint?
- Nothing. This app needs no secrets.

## Check after the deploy — the coding agent

- [ ] `builder init https://github.com/example/go-hello` completes and produces a workspace directory with a builder.yaml and an artifact zip
- [ ] The artifact directory contains both .json and .yaml metadata files alongside the zip
- [ ] The job pod exits with exit code 0 and logs show 'Build Complete 🔨'
- [ ] The volume at /data retains the artifact zip after the pod completes (check volume contents before reschedule)

---

Machine-readable: `.nexlayer/findings.json`.
