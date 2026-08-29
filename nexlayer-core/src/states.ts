export type CoreState =
  | "idle"
  | "hover"
  | "launch"
  | "deploying"
  | "scaling"
  | "success"
  | "error";

export const STATE_ORDER: readonly CoreState[] = [
  "idle",
  "hover",
  "launch",
  "deploying",
  "scaling",
  "success",
  "error",
] as const;

export const STATE_INDEX: Record<CoreState, number> = {
  idle: 0,
  hover: 1,
  launch: 2,
  deploying: 3,
  scaling: 4,
  success: 5,
  error: 6,
};

export interface StateRuntime {
  readonly state: CoreState;
  readonly progress: number;
  readonly launchPhase: number;
  readonly shockwave: number;
  readonly chaos: number;
  readonly edgeSharpness: number;
}

const STATE_DURATIONS: Partial<Record<CoreState, number>> = {
  launch: 2.2,
  deploying: 2.8,
  scaling: 2.4,
  success: 1.8,
  error: 2.6,
};

export function createStateMachine() {
  let state: CoreState = "idle";
  let progress = 0;
  let launchPhase = 0;
  let shockwave = 0;
  let chaos = 1;
  let edgeSharpness = 1;
  let hovered = false;
  let autoReturn: CoreState = "idle";

  const setState = (next: CoreState, returnTo: CoreState = "idle") => {
    state = next;
    progress = 0;
    launchPhase = 0;
    shockwave = 0;
    autoReturn = returnTo;
    if (next === "error") chaos = 1;
    if (next === "success") edgeSharpness = 1.4;
    if (next === "idle") edgeSharpness = 1;
  };

  const setHovered = (value: boolean) => {
    hovered = value;
    if (state === "idle" && value) state = "hover";
    if (state === "hover" && !value) state = "idle";
  };

  const update = (dt: number): StateRuntime => {
    const duration = STATE_DURATIONS[state];
    if (duration !== undefined) {
      progress = Math.min(1, progress + dt / duration);

      if (state === "launch") {
        launchPhase = progress;
      }
      if (state === "success") {
        shockwave = progress;
        edgeSharpness = 1 + (1 - progress) * 0.6;
      }
      if (state === "error") {
        chaos = 1 - progress;
      }

      if (progress >= 1) {
        setState(autoReturn);
      }
    }

    if (state === "idle" || state === "hover") {
      const breathe = Math.sin(performance.now() * 0.0015) * 0.5 + 0.5;
      edgeSharpness = 0.92 + breathe * 0.08;
    }

    if (hovered && (state === "idle" || state === "hover")) {
      state = "hover";
    }

    return {
      state,
      progress,
      launchPhase,
      shockwave,
      chaos,
      edgeSharpness,
    };
  };

  return { setState, setHovered, update, getState: () => state };
}

export type StateMachine = ReturnType<typeof createStateMachine>;
