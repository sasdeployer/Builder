import { createRenderer } from "./renderer";
import { STATE_ORDER, type CoreState } from "./states";

const canvas = document.querySelector<HTMLCanvasElement>("#core");
const badge = document.querySelector<HTMLDivElement>("#state-badge");
const buttonContainer = document.querySelector<HTMLDivElement>("#state-buttons");

if (!canvas || !badge || !buttonContainer) {
  throw new Error("Missing required DOM elements");
}

const renderer = createRenderer(canvas);
let activeState: CoreState = "idle";

const labels: Record<CoreState, string> = {
  idle: "Idle — calm circulation",
  hover: "Hover — magnetic rerouting",
  launch: "Launch — deconstruct & reform",
  deploying: "Deploying — packet routing",
  scaling: "Scaling — ghost nodes",
  success: "Success — convergence pulse",
  error: "Error — detect & reroute",
};

function setBadge(state: CoreState) {
  badge!.textContent = state;
  badge!.style.color = state === "error" ? "#ff6b6b" : "#00e5d4";
}

function updateButtons(state: CoreState) {
  for (const btn of buttonContainer!.querySelectorAll<HTMLButtonElement>(".state-btn")) {
    btn.classList.toggle("active", btn.dataset.state === state);
  }
}

for (const state of STATE_ORDER) {
  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = "state-btn";
  btn.dataset.state = state;
  btn.textContent = labels[state];
  btn.addEventListener("click", () => {
    activeState = state;
    renderer.setState(state);
    setBadge(state);
    updateButtons(state);
  });
  buttonContainer.appendChild(btn);
}

canvas.addEventListener("click", () => {
  activeState = "launch";
  renderer.setState("launch");
  setBadge("launch");
  updateButtons("launch");
});

canvas.addEventListener("pointerenter", () => {
  if (activeState === "idle") {
    renderer.setState("hover");
    setBadge("hover");
    updateButtons("hover");
  }
});

canvas.addEventListener("pointerleave", () => {
  if (activeState === "hover") {
    activeState = "idle";
    renderer.setState("idle");
    setBadge("idle");
    updateButtons("idle");
  }
});

setBadge("idle");
updateButtons("idle");
void renderer.ready;
window.addEventListener("beforeunload", () => renderer.dispose());
