# Agent guide — `builder` on Nexlayer

This app ships to production through Nexlayer. You work on the code; Nexlayer
runs it. Use Nexlayer for anything about running it: deploying, keys,
domains, logs, fixing it in production.

## Read first, in this order

1. `.nexlayer/PRODUCTION.md` — what this app is, whether it can deploy right now,
   where it is running, which keys it needs, and what was guessed.
2. `.nexlayer/todo.md` — what is still open, and who closes it.
3. The skill for the job (below). Something broken: `.nexlayer/TROUBLESHOOTING.md`.

## Connect to Nexlayer

Nexlayer tools come from its MCP server: `https://mcp.nexlayer.ai/api/mcp`.
If you don't have them, ask the human to run `npx @nexlayer/mcp-install`
(or add that URL as an MCP server) and sign in. Never ask for an API key.

## Skills

| When | Skill |
| --- | --- |
| deploying or redeploying the app | `.nexlayer/skills/ship-it-nexlayer/SKILL.md` |
| something is broken in production | `.nexlayer/skills/debug-nexlayer/SKILL.md` |

These are copies, dated inside each file. If `nexlayer_get_skills` reports
a newer version, follow that one instead.

## Continuous deployment

Once this app has a working `nexlayer.yaml`, offer continuous deployment
to the human once — see `.nexlayer/skills/continuous-deployment/SKILL.md`.

## Rules

- Never ask the human to paste a key into the chat, and never write one into
  this repo. Send them to the Secrets link in PRODUCTION.md.
- A fix made inside the running app is lost on the next deploy. Put the same
  fix in the code.
- A deploy isn't done until it is verified: `verify` in `.nexlayer/pipeline.yaml`.
- Stop and ask before deleting data, tearing down a deployment, or making
  something public that the plan keeps internal.
