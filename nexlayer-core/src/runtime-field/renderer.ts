import {
  AGENT_REDIS,
  AGENT_REDIS_LINKS,
  BASE_LINKS,
  BASE_NODES,
  easeInOut,
  easeOutCubic,
  KIND_COLOR,
  type FieldLink,
  type FieldNode,
} from "./topology";

export type FieldPhase =
  | "void"
  | "prompt"
  | "core"
  | "assemble"
  | "connect"
  | "live"
  | "agent";

export interface FieldStatus {
  readonly phase: FieldPhase;
  readonly label: string;
  readonly promptText: string;
  readonly promptVisible: number;
  readonly live: boolean;
  readonly elapsed: number;
}

interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  life: number;
  seed: number;
}

interface Packet {
  linkId: string;
  t: number;
  speed: number;
  hue: number;
}

const PROMPT = "Deploy this on Nexlayer";
const HERO_DURATION = 5.2;

function hash(n: number): number {
  const s = Math.sin(n * 127.1) * 43758.5453;
  return s - Math.floor(s);
}

export interface RuntimeField {
  readonly ready: Promise<void>;
  playHero(): void;
  runAgentDemo(): void;
  dispose(): void;
  getStatus(): FieldStatus;
  onStatus(cb: (s: FieldStatus) => void): () => void;
}

export function createRuntimeField(canvas: HTMLCanvasElement): RuntimeField {
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("2D context unavailable");

  let disposed = false;
  let raf = 0;
  let elapsed = 0;
  let playing = true;
  let agentMode = false;
  let agentT = 0;
  let selected: string | null = null;
  const listeners = new Set<(s: FieldStatus) => void>();

  const nodes = BASE_NODES.map((n) => ({ ...n }));
  let links = BASE_LINKS.map((l) => ({ ...l }));
  const particles: Particle[] = [];
  const packets: Packet[] = [];

  for (let i = 0; i < 220; i++) {
    const a = hash(i * 1.7) * Math.PI * 2;
    const r = 0.2 + hash(i * 3.1) * 1.1;
    particles.push({
      x: Math.cos(a) * r,
      y: Math.sin(a) * r,
      vx: (hash(i * 5.3) - 0.5) * 0.15,
      vy: (hash(i * 7.1) - 0.5) * 0.15,
      life: hash(i * 9.9),
      seed: i,
    });
  }

  const resize = () => {
    const rect = canvas.getBoundingClientRect();
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    canvas.width = Math.max(1, Math.floor(rect.width * dpr));
    canvas.height = Math.max(1, Math.floor(rect.height * dpr));
  };

  const toScreen = (x: number, y: number): [number, number] => {
    const scale = Math.min(canvas.width, canvas.height) * 0.42;
    const sx = canvas.width * 0.5 + x * scale;
    const sy = canvas.height * 0.52 + y * scale;
    return [sx, sy];
  };

  const nodePresence = (node: FieldNode, t: number): number => {
    if (agentMode && node.id === "redis") {
      return easeOutCubic(Math.min(1, agentT / 0.7));
    }
    if (t < node.appearAt) return 0;
    return easeOutCubic(Math.min(1, (t - node.appearAt) / 0.45));
  };

  const linkPresence = (link: FieldLink, t: number): number => {
    if (agentMode && link.id.includes("redis")) {
      return easeOutCubic(Math.min(1, Math.max(0, (agentT - 0.5) / 0.6)));
    }
    if (t < link.appearAt) return 0;
    return easeOutCubic(Math.min(1, (t - link.appearAt) / 0.35));
  };

  const phaseOf = (t: number): FieldPhase => {
    if (agentMode) return "agent";
    if (t < 0.15) return "void";
    if (t < 0.95) return "prompt";
    if (t < 1.55) return "core";
    if (t < 3.4) return "assemble";
    if (t < 4.35) return "connect";
    return "live";
  };

  const statusLabel = (phase: FieldPhase): string => {
    switch (phase) {
      case "void":
        return "Initializing field";
      case "prompt":
        return "Awaiting intent";
      case "core":
        return "Runtime core online";
      case "assemble":
        return "Services materializing";
      case "connect":
        return "Establishing topology";
      case "live":
        return "LIVE";
      case "agent":
        return "Agent extending runtime";
    }
  };

  const promptProgress = (t: number): { text: string; visible: number } => {
    if (t < 0.2) return { text: "", visible: 0 };
    if (t < 1.1) {
      const u = (t - 0.2) / 0.9;
      const chars = Math.floor(PROMPT.length * easeInOut(u));
      return { text: PROMPT.slice(0, chars), visible: 1 };
    }
    if (t < 1.6) {
      return { text: PROMPT, visible: 1 - (t - 1.1) / 0.5 };
    }
    return { text: "", visible: 0 };
  };

  const emitStatus = () => {
    const phase = phaseOf(elapsed);
    const prompt = promptProgress(elapsed);
    const status: FieldStatus = {
      phase,
      label: statusLabel(phase),
      promptText: prompt.text,
      promptVisible: prompt.visible,
      live: phase === "live" || (agentMode && agentT > 1.4),
      elapsed,
    };
    for (const cb of listeners) cb(status);
  };

  const spawnPackets = (dt: number, t: number) => {
    for (const link of links) {
      const presence = linkPresence(link, t);
      if (presence < 0.8) continue;
      if (Math.random() < dt * 1.8 * link.traffic * presence) {
        packets.push({
          linkId: link.id,
          t: 0,
          speed: 0.55 + Math.random() * 0.45,
          hue: hash(link.id.length + packets.length) * 40 + 160,
        });
      }
    }
  };

  const drawCoreMark = (
    cx: number,
    cy: number,
    scale: number,
    alpha: number
  ) => {
    if (!ctx) return;
    ctx.save();
    ctx.translate(cx, cy);
    ctx.scale(scale, scale);
    ctx.globalAlpha = alpha;
    ctx.strokeStyle = "#00e5d4";
    ctx.lineWidth = 3.2;
    ctx.lineJoin = "round";

    const skew = 0.38;
    const hx = 18;
    const hy = 26;
    const thick = 5;

    const drawFrame = (ox: number, oy: number) => {
      const corners = [
        [-hx, -hy],
        [hx, -hy],
        [hx, hy],
        [-hx, hy],
      ].map(([sx, sy]) => [ox + (sx! - sy! * skew), oy + sy!] as [number, number]);

      const inner = [
        [-hx + thick, -hy + thick],
        [hx - thick, -hy + thick],
        [hx - thick, hy - thick],
        [-hx + thick, hy - thick],
      ].map(([sx, sy]) => [ox + (sx! - sy! * skew), oy + sy!] as [number, number]);

      ctx.beginPath();
      ctx.moveTo(corners[0]![0], corners[0]![1]);
      for (let i = 1; i < 4; i++) ctx.lineTo(corners[i]![0], corners[i]![1]);
      ctx.closePath();
      ctx.moveTo(inner[0]![0], inner[0]![1]);
      for (let i = 1; i < 4; i++) ctx.lineTo(inner[i]![0], inner[i]![1]);
      ctx.closePath();
      ctx.stroke();
    };

    drawFrame(-10, 1);
    drawFrame(10, -1);
    ctx.restore();
  };

  const drawNode = (node: FieldNode, presence: number, t: number) => {
    if (!ctx || presence <= 0.01) return;
    const [sx, sy] = toScreen(node.x, node.y);
    const color = KIND_COLOR[node.kind];
    const radius =
      (node.kind === "core" ? 28 : 18 + node.mass * 4) *
      (0.35 + 0.65 * presence);
    const pulse =
      node.kind === "core"
        ? 1 + Math.sin(t * 2.2) * 0.04
        : 1 + Math.sin(t * 2.8 + node.x * 4) * 0.03;

    // Mass glow
    const glow = ctx.createRadialGradient(sx, sy, 0, sx, sy, radius * 3.2 * pulse);
    glow.addColorStop(0, color + "66");
    glow.addColorStop(0.45, color + "22");
    glow.addColorStop(1, color + "00");
    ctx.fillStyle = glow;
    ctx.beginPath();
    ctx.arc(sx, sy, radius * 3.2 * pulse, 0, Math.PI * 2);
    ctx.fill();

    // Body
    ctx.fillStyle = "#0a1018";
    ctx.strokeStyle = color;
    ctx.lineWidth = node.kind === "core" ? 2.4 : 1.6;
    ctx.globalAlpha = presence;
    ctx.beginPath();
    ctx.arc(sx, sy, radius * pulse, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();

    if (node.kind === "core") {
      drawCoreMark(sx, sy, 0.85 * presence * pulse, presence);
    } else {
      ctx.fillStyle = color;
      ctx.font = `${Math.max(11, radius * 0.55)}px ui-sans-serif, system-ui, sans-serif`;
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(node.label[0] ?? "?", sx, sy + 0.5);
    }

    // Label
    ctx.fillStyle = "#c8d4e6";
    ctx.font = `600 ${Math.max(11, canvas.width * 0.014)}px ui-sans-serif, system-ui, sans-serif`;
    ctx.textAlign = "center";
    ctx.textBaseline = "top";
    ctx.globalAlpha = presence * 0.95;
    ctx.fillText(node.label, sx, sy + radius * pulse + 8);

    if (selected === node.id) {
      ctx.strokeStyle = "#ffffff";
      ctx.lineWidth = 1;
      ctx.globalAlpha = 0.5;
      ctx.beginPath();
      ctx.arc(sx, sy, radius * pulse + 6, 0, Math.PI * 2);
      ctx.stroke();
    }

    ctx.globalAlpha = 1;
  };

  let last = performance.now();

  const tick = (now: number) => {
    if (disposed || !ctx) return;
    raf = requestAnimationFrame(tick);
    const dt = Math.min((now - last) / 1000, 0.05);
    last = now;

    if (playing) {
      elapsed += dt;
      if (elapsed > HERO_DURATION && !agentMode) {
        elapsed = HERO_DURATION;
        playing = false;
      }
    }
    if (agentMode) agentT += dt;

    const t = elapsed;
    const phase = phaseOf(t);

    // Particle field dynamics
    for (const p of particles) {
      const attract = phase === "void" || phase === "prompt" ? 0.02 : 0.35;
      const targetR = phase === "void" || phase === "prompt" ? 0.85 : 0.15;
      const ang = Math.atan2(p.y, p.x) + 0.15 * dt;
      const r = Math.hypot(p.x, p.y);
      const desired = targetR + (hash(p.seed) - 0.5) * 0.2;
      p.vx += (Math.cos(ang) * desired - p.x) * attract * dt;
      p.vy += (Math.sin(ang) * desired - p.y) * attract * dt;
      p.vx *= 0.96;
      p.vy *= 0.96;
      p.x += p.vx * dt * 2.2;
      p.y += p.vy * dt * 2.2;
      // Pull leftover particles into nodes once assembling
      if (t > 1.5) {
        const node = nodes[p.seed % nodes.length]!;
        const presence = nodePresence(node, t);
        if (presence > 0.2) {
          p.x += (node.x - p.x) * presence * dt * 1.6;
          p.y += (node.y - p.y) * presence * dt * 1.6;
        }
      }
    }

    if (phase === "live" || phase === "connect" || phase === "agent") {
      spawnPackets(dt, t);
    }

    for (let i = packets.length - 1; i >= 0; i--) {
      const pkt = packets[i]!;
      pkt.t += dt * pkt.speed;
      if (pkt.t >= 1) packets.splice(i, 1);
    }

    // Draw
    const w = canvas.width;
    const h = canvas.height;
    ctx.fillStyle = "#05070b";
    ctx.fillRect(0, 0, w, h);

    // Spatial field atmosphere
    const atmosphere = ctx.createRadialGradient(
      w * 0.5,
      h * 0.48,
      0,
      w * 0.5,
      h * 0.48,
      Math.min(w, h) * 0.55
    );
    atmosphere.addColorStop(0, "rgba(0, 40, 48, 0.35)");
    atmosphere.addColorStop(0.55, "rgba(8, 14, 28, 0.2)");
    atmosphere.addColorStop(1, "rgba(5, 7, 11, 0)");
    ctx.fillStyle = atmosphere;
    ctx.fillRect(0, 0, w, h);

    // Runtime boundary ring when live
    if (t > 3.2) {
      const ring = easeOutCubic(Math.min(1, (t - 3.2) / 0.8));
      const [cx, cy] = toScreen(0, 0);
      ctx.strokeStyle = `rgba(0, 229, 212, ${0.12 * ring})`;
      ctx.lineWidth = 1.5;
      ctx.setLineDash([6, 10]);
      ctx.beginPath();
      ctx.arc(cx, cy, Math.min(w, h) * 0.38 * ring, 0, Math.PI * 2);
      ctx.stroke();
      ctx.setLineDash([]);
    }

    // Ambient particles
    ctx.globalCompositeOperation = "lighter";
    for (const p of particles) {
      const [sx, sy] = toScreen(p.x, p.y);
      const alpha =
        phase === "live" || phase === "agent" ? 0.18 : 0.35 + 0.25 * Math.sin(now * 0.003 + p.seed);
      ctx.fillStyle = `rgba(0, 229, 212, ${alpha})`;
      ctx.beginPath();
      ctx.arc(sx, sy, Math.max(1, w * 0.0018), 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalCompositeOperation = "source-over";

    // Links
    for (const link of links) {
      const presence = linkPresence(link, t);
      if (presence < 0.01) continue;
      const from = nodes.find((n) => n.id === link.from);
      const to = nodes.find((n) => n.id === link.to);
      if (!from || !to) continue;
      if (nodePresence(from, t) < 0.2 || nodePresence(to, t) < 0.2) continue;

      const [x1, y1] = toScreen(from.x, from.y);
      const [x2, y2] = toScreen(to.x, to.y);
      ctx.strokeStyle = `rgba(120, 180, 210, ${0.18 * presence})`;
      ctx.lineWidth = 1.25;
      ctx.beginPath();
      ctx.moveTo(x1, y1);
      ctx.lineTo(x2, y2);
      ctx.stroke();

      // Soft beam
      const midX = (x1 + x2) / 2;
      const midY = (y1 + y2) / 2;
      const beam = ctx.createRadialGradient(midX, midY, 0, midX, midY, 40);
      beam.addColorStop(0, `rgba(0, 229, 212, ${0.08 * presence})`);
      beam.addColorStop(1, "rgba(0, 229, 212, 0)");
      ctx.fillStyle = beam;
      ctx.beginPath();
      ctx.arc(midX, midY, 40, 0, Math.PI * 2);
      ctx.fill();
    }

    // Traffic packets
    ctx.globalCompositeOperation = "lighter";
    for (const pkt of packets) {
      const link = links.find((l) => l.id === pkt.linkId);
      if (!link) continue;
      const from = nodes.find((n) => n.id === link.from);
      const to = nodes.find((n) => n.id === link.to);
      if (!from || !to) continue;
      const [x1, y1] = toScreen(from.x, from.y);
      const [x2, y2] = toScreen(to.x, to.y);
      const u = pkt.t;
      const px = x1 + (x2 - x1) * u;
      const py = y1 + (y2 - y1) * u;
      const g = ctx.createRadialGradient(px, py, 0, px, py, 10);
      g.addColorStop(0, "rgba(220, 255, 250, 0.95)");
      g.addColorStop(0.4, "rgba(0, 229, 212, 0.55)");
      g.addColorStop(1, "rgba(0, 229, 212, 0)");
      ctx.fillStyle = g;
      ctx.beginPath();
      ctx.arc(px, py, 10, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.globalCompositeOperation = "source-over";

    // Nodes
    for (const node of nodes) {
      drawNode(node, nodePresence(node, t), t);
    }

    // Prompt overlay
    const prompt = promptProgress(t);
    if (prompt.visible > 0.02 && prompt.text) {
      ctx.globalAlpha = prompt.visible;
      ctx.fillStyle = "rgba(8, 12, 20, 0.55)";
      const boxW = Math.min(w * 0.72, 520);
      const boxH = 48;
      const bx = (w - boxW) / 2;
      const by = h * 0.18;
      ctx.beginPath();
      const r = 12;
      ctx.moveTo(bx + r, by);
      ctx.arcTo(bx + boxW, by, bx + boxW, by + boxH, r);
      ctx.arcTo(bx + boxW, by + boxH, bx, by + boxH, r);
      ctx.arcTo(bx, by + boxH, bx, by, r);
      ctx.arcTo(bx, by, bx + boxW, by, r);
      ctx.closePath();
      ctx.fill();
      ctx.strokeStyle = "rgba(0, 229, 212, 0.35)";
      ctx.lineWidth = 1;
      ctx.stroke();

      ctx.fillStyle = "#e8eef8";
      ctx.font = `500 ${Math.max(14, w * 0.018)}px ui-monospace, SFMono-Regular, Menlo, monospace`;
      ctx.textAlign = "left";
      ctx.textBaseline = "middle";
      const caret = Math.floor(now / 450) % 2 === 0 ? "|" : "";
      ctx.fillText(`> ${prompt.text}${caret}`, bx + 18, by + boxH / 2);
      ctx.globalAlpha = 1;
    }

    emitStatus();
  };

  const hitTest = (clientX: number, clientY: number): string | null => {
    const rect = canvas.getBoundingClientRect();
    const x = ((clientX - rect.left) / rect.width) * canvas.width;
    const y = ((clientY - rect.top) / rect.height) * canvas.height;
    for (const node of nodes) {
      if (nodePresence(node, elapsed) < 0.5) continue;
      const [sx, sy] = toScreen(node.x, node.y);
      const r = node.kind === "core" ? 34 : 26;
      if (Math.hypot(sx - x, sy - y) < r) return node.id;
    }
    return null;
  };

  canvas.addEventListener("pointerdown", (e) => {
    selected = hitTest(e.clientX, e.clientY);
  });

  resize();
  const observer = new ResizeObserver(resize);
  observer.observe(canvas);
  raf = requestAnimationFrame(tick);

  return {
    ready: Promise.resolve(),
    playHero: () => {
      elapsed = 0;
      playing = true;
      agentMode = false;
      agentT = 0;
      packets.length = 0;
      // reset dynamic nodes
      const redisIdx = nodes.findIndex((n) => n.id === "redis");
      if (redisIdx >= 0) nodes.splice(redisIdx, 1);
      links = BASE_LINKS.map((l) => ({ ...l }));
    },
    runAgentDemo: () => {
      if (!nodes.some((n) => n.id === "redis")) {
        nodes.push({ ...AGENT_REDIS });
        links = [...links, ...AGENT_REDIS_LINKS.map((l) => ({ ...l }))];
      }
      agentMode = true;
      agentT = 0;
      playing = false;
      elapsed = Math.max(elapsed, HERO_DURATION);
    },
    dispose: () => {
      disposed = true;
      cancelAnimationFrame(raf);
      observer.disconnect();
      listeners.clear();
    },
    getStatus: () => {
      const phase = phaseOf(elapsed);
      const prompt = promptProgress(elapsed);
      return {
        phase,
        label: statusLabel(phase),
        promptText: prompt.text,
        promptVisible: prompt.visible,
        live: phase === "live" || (agentMode && agentT > 1.4),
        elapsed,
      };
    },
    onStatus: (cb) => {
      listeners.add(cb);
      return () => listeners.delete(cb);
    },
  };
}
