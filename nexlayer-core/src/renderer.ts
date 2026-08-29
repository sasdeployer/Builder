import { createCanvasSwarm, type CoreRenderer } from "./canvas-swarm";
import type { CoreState } from "./states";

export type { CoreRenderer };

/**
 * Living Nexlayer Core renderer.
 * Uses a Canvas2D particle swarm that assembles the Nexlayer mark
 * (two overlapping parallelogram frames). WGSL/vGPU shaders live in
 * `src/shaders/` for WebGPU environments; this path guarantees the
 * swarm-forms-icon visual everywhere.
 */
export function createRenderer(canvas: HTMLCanvasElement): CoreRenderer {
  return createCanvasSwarm(canvas);
}

export type { CoreState };
