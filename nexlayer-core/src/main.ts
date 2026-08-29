import { createRuntimeField } from "./runtime-field/renderer";
import type { FieldPhase } from "./runtime-field/renderer";

const canvas = document.querySelector<HTMLCanvasElement>("#field");
const statusLabel = document.querySelector<HTMLSpanElement>("#status-label");
const statusChip = document.querySelector<HTMLDivElement>("#status-chip");
const btnReplay = document.querySelector<HTMLButtonElement>("#btn-replay");
const btnAgent = document.querySelector<HTMLButtonElement>("#btn-agent");
const beatList = document.querySelectorAll<HTMLLIElement>("#beat-list li");

if (!canvas || !statusLabel || !statusChip || !btnReplay || !btnAgent) {
  throw new Error("Missing Runtime Field DOM nodes");
}

const field = createRuntimeField(canvas);

const phaseToBeat: Record<FieldPhase, string> = {
  void: "prompt",
  prompt: "prompt",
  core: "core",
  assemble: "assemble",
  connect: "connect",
  live: "live",
  agent: "agent",
};

const beatOrder = ["prompt", "core", "assemble", "connect", "live", "agent"];

field.onStatus((status) => {
  statusLabel.textContent = status.label;
  statusChip.classList.toggle("live", status.live);

  const activeBeat = phaseToBeat[status.phase];
  const activeIndex = beatOrder.indexOf(activeBeat);

  for (const li of beatList) {
    const beat = li.dataset.beat ?? "";
    const idx = beatOrder.indexOf(beat);
    li.classList.toggle("active", beat === activeBeat);
    li.classList.toggle("done", idx >= 0 && idx < activeIndex);
  }
});

btnReplay.addEventListener("click", () => {
  field.playHero();
});

btnAgent.addEventListener("click", () => {
  field.runAgentDemo();
});

void field.ready;

window.addEventListener("beforeunload", () => field.dispose());
