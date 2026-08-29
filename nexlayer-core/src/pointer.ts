export interface PointerState {
  readonly position: readonly [number, number];
  readonly active: boolean;
  readonly inside: boolean;
}

export function installPointer(canvas: HTMLCanvasElement): PointerState & { dispose(): void } {
  let position: [number, number] = [0.5, 0.5];
  let active = false;
  let inside = false;
  const previousTouchAction = canvas.style.touchAction;
  canvas.style.touchAction = "none";

  const toUv = (event: PointerEvent): [number, number] => {
    const rect = canvas.getBoundingClientRect();
    return [
      Math.max(0, Math.min(1, (event.clientX - rect.left) / Math.max(1, rect.width))),
      Math.max(0, Math.min(1, (event.clientY - rect.top) / Math.max(1, rect.height))),
    ];
  };

  const onEnter = (event: PointerEvent) => {
    inside = true;
    position = toUv(event);
    active = true;
  };

  const onMove = (event: PointerEvent) => {
    position = toUv(event);
    active = true;
  };

  const onLeave = () => {
    inside = false;
    active = false;
  };

  const onDown = (event: PointerEvent) => {
    position = toUv(event);
    active = true;
  };

  canvas.addEventListener("pointerenter", onEnter);
  canvas.addEventListener("pointermove", onMove);
  canvas.addEventListener("pointerdown", onDown);
  canvas.addEventListener("pointerleave", onLeave);
  canvas.addEventListener("pointercancel", onLeave);

  return {
    get position() {
      return position;
    },
    get active() {
      return active && inside;
    },
    get inside() {
      return inside;
    },
    dispose() {
      canvas.removeEventListener("pointerenter", onEnter);
      canvas.removeEventListener("pointermove", onMove);
      canvas.removeEventListener("pointerdown", onDown);
      canvas.removeEventListener("pointerleave", onLeave);
      canvas.removeEventListener("pointercancel", onLeave);
      canvas.style.touchAction = previousTouchAction;
    },
  };
}
