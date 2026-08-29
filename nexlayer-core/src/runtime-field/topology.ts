export type NodeKind =
  | "core"
  | "api"
  | "frontend"
  | "postgres"
  | "worker"
  | "external"
  | "storage"
  | "cache";

export interface FieldNode {
  readonly id: string;
  readonly label: string;
  readonly kind: NodeKind;
  /** Resting position in field space (-1..1) */
  readonly x: number;
  readonly y: number;
  readonly mass: number;
  /** When this node begins materializing in the hero timeline (seconds) */
  readonly appearAt: number;
}

export interface FieldLink {
  readonly id: string;
  readonly from: string;
  readonly to: string;
  readonly appearAt: number;
  readonly traffic: number;
}

export const BASE_NODES: readonly FieldNode[] = [
  {
    id: "nexlayer",
    label: "NEXLAYER",
    kind: "core",
    x: 0,
    y: 0,
    mass: 1.4,
    appearAt: 0.9,
  },
  {
    id: "api",
    label: "API",
    kind: "api",
    x: -0.32,
    y: 0.02,
    mass: 1,
    appearAt: 1.55,
  },
  {
    id: "frontend",
    label: "Frontend",
    kind: "frontend",
    x: 0.02,
    y: 0.58,
    mass: 0.9,
    appearAt: 2.15,
  },
  {
    id: "postgres",
    label: "Postgres",
    kind: "postgres",
    x: -0.68,
    y: 0.08,
    mass: 1.2,
    appearAt: 2.35,
  },
  {
    id: "worker",
    label: "Worker",
    kind: "worker",
    x: 0.55,
    y: 0.05,
    mass: 0.95,
    appearAt: 2.85,
  },
  {
    id: "openrouter",
    label: "OpenRouter",
    kind: "external",
    x: 0.08,
    y: -0.62,
    mass: 0.85,
    appearAt: 3.1,
  },
];

export const BASE_LINKS: readonly FieldLink[] = [
  { id: "nl-api", from: "nexlayer", to: "api", appearAt: 3.45, traffic: 1 },
  { id: "api-pg", from: "api", to: "postgres", appearAt: 3.55, traffic: 0.85 },
  { id: "nl-fe", from: "nexlayer", to: "frontend", appearAt: 3.65, traffic: 0.9 },
  { id: "api-fe", from: "api", to: "frontend", appearAt: 3.7, traffic: 0.7 },
  { id: "nl-worker", from: "nexlayer", to: "worker", appearAt: 3.8, traffic: 0.8 },
  { id: "nl-or", from: "nexlayer", to: "openrouter", appearAt: 3.9, traffic: 0.75 },
  { id: "worker-or", from: "worker", to: "openrouter", appearAt: 4.0, traffic: 0.6 },
];

export const AGENT_REDIS: FieldNode = {
  id: "redis",
  label: "Redis",
  kind: "cache",
  x: -0.55,
  y: -0.38,
  mass: 1.05,
  appearAt: 0,
};

export const AGENT_REDIS_LINKS: readonly FieldLink[] = [
  { id: "api-redis", from: "api", to: "redis", appearAt: 0, traffic: 0.95 },
  { id: "worker-redis", from: "worker", to: "redis", appearAt: 0, traffic: 0.7 },
];

export const KIND_COLOR: Record<NodeKind, string> = {
  core: "#00e5d4",
  api: "#5b8cff",
  frontend: "#7dd3fc",
  postgres: "#34d399",
  worker: "#a78bfa",
  external: "#fbbf24",
  storage: "#94a3b8",
  cache: "#fb7185",
};

export function easeOutCubic(t: number): number {
  const x = Math.min(1, Math.max(0, t));
  return 1 - (1 - x) ** 3;
}

export function easeInOut(t: number): number {
  const x = Math.min(1, Math.max(0, t));
  return x < 0.5 ? 2 * x * x : 1 - (-2 * x + 2) ** 2 / 2;
}
