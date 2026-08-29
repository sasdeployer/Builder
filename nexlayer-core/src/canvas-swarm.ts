import { createStateMachine, type CoreState } from "./states";
import { installPointer } from "./pointer";

const PARTICLE_COUNT = 900;

interface Particle {
  edge: number;
  along: number;
  across: number;
  phase: number;
  x: number;
  y: number;
}

const FRAME_HALF = { x: 0.195, y: 0.305 };
const FRAME_SKEW = 0.38;
const FRAME_THICK = 0.048;
const LEFT = { x: -0.155, y: 0.015 };
const RIGHT = { x: 0.155, y: -0.015 };

function unskew(sx: number, sy: number): [number, number] {
  return [sx - sy * FRAME_SKEW, sy];
}

function frameCorner(frame: number, corner: number): [number, number] {
  const center = frame === 0 ? LEFT : RIGHT;
  const hx = FRAME_HALF.x - FRAME_THICK * 0.5;
  const hy = FRAME_HALF.y - FRAME_THICK * 0.5;
  const corners: Array<[number, number]> = [
    [-hx, -hy],
    [hx, -hy],
    [hx, hy],
    [-hx, hy],
  ];
  const [sx, sy] = corners[corner]!;
  const [ux, uy] = unskew(sx, sy);
  return [center.x + ux, center.y + uy];
}

function edgePoint(edge: number, t: number): [number, number] {
  const frame = Math.floor(edge / 4);
  const side = edge % 4;
  const a = frameCorner(frame, side);
  const b = frameCorner(frame, (side + 1) % 4);
  const u = t - Math.floor(t);
  return [a[0] + (b[0] - a[0]) * u, a[1] + (b[1] - a[1]) * u];
}

function particleHome(p: Particle, flow: number): [number, number] {
  const along = (p.along + flow) % 1;
  const [bx, by] = edgePoint(p.edge, along);
  const a = frameCorner(Math.floor(p.edge / 4), p.edge % 4);
  const b = frameCorner(Math.floor(p.edge / 4), (p.edge % 4 + 1) % 4);
  const tx = b[0] - a[0];
  const ty = b[1] - a[1];
  const len = Math.hypot(tx, ty) || 1;
  const nx = -ty / len;
  const ny = tx / len;
  return [bx + nx * p.across, by + ny * p.across];
}

function hash(n: number): number {
  const s = Math.sin(n * 127.1) * 43758.5453;
  return s - Math.floor(s);
}

export interface CoreRenderer {
  readonly ready: Promise<void>;
  setState(state: CoreState): void;
  dispose(): void;
}

export function createCanvasSwarm(canvas: HTMLCanvasElement): CoreRenderer {
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("2D canvas unavailable");

  let disposed = false;
  let raf = 0;
  const pointer = installPointer(canvas);
  const stateMachine = createStateMachine();
  const particles: Particle[] = [];

  for (let i = 0; i < PARTICLE_COUNT; i++) {
    const edge = i % 8;
    const along = (i * 0.6180339887) % 1;
    const across = (hash(i * 0.017) - 0.5) * FRAME_THICK * 0.85;
    const phase = hash(i * 2.71);
    const [x, y] = particleHome(
      { edge, along, across, phase, x: 0, y: 0 },
      0
    );
    particles.push({
      edge,
      along,
      across,
      phase,
      x: x + (hash(i * 3.1) - 0.5) * 0.5,
      y: y + (hash(i * 4.7) - 0.5) * 0.5,
    });
  }

  const resize = () => {
    const rect = canvas.getBoundingClientRect();
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    canvas.width = Math.max(1, Math.floor(rect.width * dpr));
    canvas.height = Math.max(1, Math.floor(rect.height * dpr));
  };

  const toScreen = (x: number, y: number): [number, number] => {
    const aspect = canvas.width / Math.max(canvas.height, 1);
    const sx = ((x / 0.85) / aspect) * 0.5 + 0.5;
    const sy = (y / 0.85) * 0.5 + 0.5;
    return [sx * canvas.width, sy * canvas.height];
  };

  let last = performance.now();

  const tick = (now: number) => {
    if (disposed || !ctx) return;
    raf = requestAnimationFrame(tick);
    const dt = Math.min((now - last) / 1000, 0.05);
    last = now;
    const runtime = stateMachine.update(dt);
    stateMachine.setHovered(pointer.inside);
    const t = now / 1000;

    for (let i = 0; i < particles.length; i++) {
      const p = particles[i]!;
      const flow = t * (0.04 + hash(i * 0.91) * 0.02);
      let [hx, hy] = particleHome(p, flow * 0.5);

      if (runtime.state === "hover" && pointer.active) {
        const [px, py] = [
          (pointer.position[0] * 2 - 1) * 0.85,
          (pointer.position[1] * 2 - 1) * 0.85,
        ];
        const dx = hx - px;
        const dy = hy - py;
        const dist = Math.hypot(dx, dy);
        if (dist < 0.28 && dist > 1e-5) {
          const push = (0.28 - dist) * 0.55;
          hx += (dx / dist) * push;
          hy += (dy / dist) * push;
        }
      }

      if (runtime.state === "launch") {
        const prog = runtime.progress;
        if (prog < 0.32) {
          const angle = i * 2.399963 + t * 2;
          const radius = 0.12 + hash(i) * 0.5 * (prog / 0.32);
          hx = Math.cos(angle) * radius;
          hy = Math.sin(angle) * radius;
        } else if (prog < 0.62) {
          const sub = Math.floor(hash(i * 0.53 + Math.floor(t * 3)) * 8);
          [hx, hy] = edgePoint(sub, (t * 0.4 + i * 0.001) % 1);
        } else {
          const snap = (prog - 0.62) / 0.38;
          const [ax, ay] = particleHome(p, 0);
          const angle = i * 2.399963;
          hx = Math.cos(angle) * 0.35 * (1 - snap) + ax * snap;
          hy = Math.sin(angle) * 0.35 * (1 - snap) + ay * snap;
        }
      }

      if (runtime.state === "deploying") {
        const prog = runtime.progress;
        const lane = (i % 8) / 8;
        if (prog < 0.22) {
          const u = prog / 0.22;
          const [ex, ey] = edgePoint(i % 8, 0);
          hx = -0.7 * (1 - u) + ex * u;
          hy = (lane * 0.55 - 0.27) * (1 - u) + ey * u;
        } else if (prog < 0.72) {
          [hx, hy] = particleHome(p, (prog - 0.22) / 0.5);
        } else {
          const u = (prog - 0.72) / 0.28;
          const [ex, ey] = edgePoint(i % 8, 1);
          hx = ex * (1 - u) + 0.72 * u;
          hy = ey * (1 - u);
        }
      }

      if (runtime.state === "scaling") {
        const copy = i % 4;
        const strength = Math.sin(runtime.progress * Math.PI);
        const angle = copy * 1.5708 + t * 0.25;
        const [bx, by] = particleHome(p, flow);
        const scale = 1 - strength * 0.22 * (copy > 0 ? 1 : 0);
        hx = bx * scale + Math.cos(angle) * 0.3 * strength;
        hy = by * scale + Math.sin(angle) * 0.3 * strength;
      }

      if (runtime.state === "success") {
        [hx, hy] = particleHome(p, 0);
      }

      if (runtime.state === "error") {
        const chaos = 1 - runtime.progress;
        if (chaos > 0.08) {
          [hx, hy] = particleHome(p, flow * 2.2);
          hx += Math.sin(t * 14 + i) * chaos * 0.28;
          hy += Math.cos(t * 11 + i * 1.3) * chaos * 0.28;
        } else {
          [hx, hy] = particleHome(p, 0);
        }
      }

      const spring = runtime.state === "success" ? 18 : 10;
      p.x += (hx - p.x) * Math.min(1, spring * dt);
      p.y += (hy - p.y) * Math.min(1, spring * dt);
    }

    const w = canvas.width;
    const h = canvas.height;
    ctx.fillStyle = "#06080c";
    ctx.fillRect(0, 0, w, h);

    // Soft center glow
    const grd = ctx.createRadialGradient(w / 2, h / 2, 0, w / 2, h / 2, w * 0.45);
    grd.addColorStop(0, "rgba(0, 40, 45, 0.45)");
    grd.addColorStop(1, "rgba(0, 0, 0, 0)");
    ctx.fillStyle = grd;
    ctx.fillRect(0, 0, w, h);

    ctx.globalCompositeOperation = "lighter";
    for (const p of particles) {
      const [sx, sy] = toScreen(p.x, p.y);
      const radius = Math.max(1.2, w * 0.0032);
      const g = ctx.createRadialGradient(sx, sy, 0, sx, sy, radius * 3);
      g.addColorStop(0, "rgba(180, 255, 245, 0.95)");
      g.addColorStop(0.35, "rgba(0, 229, 212, 0.55)");
      g.addColorStop(1, "rgba(0, 229, 212, 0)");
      ctx.fillStyle = g;
      ctx.beginPath();
      ctx.arc(sx, sy, radius * 3, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalCompositeOperation = "source-over";

    if (runtime.shockwave > 0 && runtime.shockwave < 1) {
      const r = runtime.shockwave * Math.min(w, h) * 0.4;
      ctx.strokeStyle = `rgba(0, 229, 212, ${1 - runtime.shockwave})`;
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.arc(w / 2, h / 2, r, 0, Math.PI * 2);
      ctx.stroke();
    }
  };

  resize();
  const observer = new ResizeObserver(resize);
  observer.observe(canvas);
  raf = requestAnimationFrame(tick);

  return {
    ready: Promise.resolve(),
    setState: (state: CoreState) => stateMachine.setState(state),
    dispose: () => {
      disposed = true;
      cancelAnimationFrame(raf);
      observer.disconnect();
      pointer.dispose();
    },
  };
}
