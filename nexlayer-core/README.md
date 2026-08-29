# Nexlayer Runtime Field

Signature visualization for Nexlayer: **software assembling itself into a living runtime.**

## What you see (~5 seconds)

1. Dark spatial field + scattered particles  
2. Prompt types: `Deploy this on Nexlayer`  
3. Nexlayer core ignites  
4. API → Frontend → Postgres → Worker → OpenRouter materialize  
5. Connections establish; traffic packets pulse  
6. Status goes **LIVE**

Then try **Claude, add Redis** — watch the topology physically grow.

## Run

```bash
cd nexlayer-core
npm install
npm run dev
```

- `/` — Runtime Field hero  
- `/icon.html` — Living Nexlayer Core (swarm icon)

## Structure

| Path | Role |
|------|------|
| `src/runtime-field/topology.ts` | Nodes, links, colors |
| `src/runtime-field/renderer.ts` | Field simulation + draw loop |
| `src/canvas-swarm.ts` | Icon particle swarm |
| `src/shaders/` | vGPU WGSL experiments for WebGPU hosts |

The visual metaphor is not containers or Kubernetes diagrams — it is **a living runtime assembling around the Nexlayer mark**.
