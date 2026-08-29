// Nexlayer logo geometry — two overlapping parallelogram frames
// Particles sample the stroke midline so the swarm IS the mark.

const FRAME_HALF: vec2f = vec2f(0.195, 0.305);
const FRAME_SKEW: f32 = 0.38;
const FRAME_THICK: f32 = 0.048;
const LEFT_CENTER: vec2f = vec2f(-0.155, 0.015);
const RIGHT_CENTER: vec2f = vec2f(0.155, -0.015);

export fn hash11(p: f32) -> f32 {
  var p3 = fract(vec3f(p) * 0.1031);
  p3 += dot(p3, p3.yzx + 33.33);
  return fract((p3.x + p3.y) * p3.z);
}

export fn hash21(p: vec2f) -> f32 {
  var p3 = fract(vec3f(p.x, p.y, p.x) * 0.1031);
  p3 += dot(p3, p3.yzx + 33.33);
  return fract((p3.x + p3.y) * p3.z);
}

export fn sdSegment(p: vec2f, a: vec2f, b: vec2f) -> f32 {
  let pa = p - a;
  let ba = b - a;
  let h = clamp(dot(pa, ba) / max(dot(ba, ba), 1e-6), 0.0, 1.0);
  return length(pa - ba * h);
}

export fn sdBox(p: vec2f, b: vec2f) -> f32 {
  let d = abs(p) - b;
  return length(max(d, vec2f(0.0))) + min(max(d.x, d.y), 0.0);
}

fn unskew(s: vec2f, skew: f32) -> vec2f {
  return vec2f(s.x - s.y * skew, s.y);
}

fn skewPoint(p: vec2f, skew: f32) -> vec2f {
  return vec2f(p.x + p.y * skew, p.y);
}

export fn sdParallelogramFrame(p: vec2f, center: vec2f, halfSize: vec2f, skew: f32, thickness: f32) -> f32 {
  let skewed = skewPoint(p - center, skew);
  let outer = sdBox(skewed, halfSize);
  let inner = sdBox(skewed, halfSize - vec2f(thickness));
  return max(outer, -inner);
}

export fn nexlayerSdf(p: vec2f) -> f32 {
  let frame1 = sdParallelogramFrame(p, LEFT_CENTER, FRAME_HALF, FRAME_SKEW, FRAME_THICK);
  let frame2 = sdParallelogramFrame(p, RIGHT_CENTER, FRAME_HALF, FRAME_SKEW, FRAME_THICK);
  return min(frame1, frame2);
}

export fn nexlayerGlow(p: vec2f, edge: f32) -> f32 {
  let d = nexlayerSdf(p);
  let stroke = 1.0 - smoothstep(0.0, 0.018, abs(d));
  let soft = 1.0 - smoothstep(0.0, 0.06, abs(d));
  return (stroke * 0.55 + soft * 0.2) * edge;
}

fn frameCorner(frame: u32, corner: u32) -> vec2f {
  let center = select(LEFT_CENTER, RIGHT_CENTER, frame == 1u);
  let half = FRAME_HALF - vec2f(FRAME_THICK * 0.5);
  var s: vec2f;
  switch corner {
    case 0u: { s = vec2f(-half.x, -half.y); }
    case 1u: { s = vec2f( half.x, -half.y); }
    case 2u: { s = vec2f( half.x,  half.y); }
    default: { s = vec2f(-half.x,  half.y); }
  }
  return center + unskew(s, FRAME_SKEW);
}

export fn iconEdgePoint(edgeId: u32, t: f32) -> vec2f {
  let frame = edgeId / 4u;
  let side = edgeId % 4u;
  let a = frameCorner(frame, side);
  let b = frameCorner(frame, (side + 1u) % 4u);
  return mix(a, b, fract(t));
}

export fn iconParticleHome(index: u32, phase: f32) -> vec2f {
  let edge = index % 8u;
  let alongFull = fract(f32(index) * 0.6180339887 + phase * 0.01);
  let base = iconEdgePoint(edge, alongFull);

  let nSeed = f32(index) * 0.017;
  let across = (hash11(nSeed) - 0.5) * FRAME_THICK * 0.85;

  let a = frameCorner(edge / 4u, edge % 4u);
  let b = frameCorner(edge / 4u, (edge % 4u + 1u) % 4u);
  let tangent = normalize(b - a);
  let normal = vec2f(-tangent.y, tangent.x);

  return base + normal * across;
}

export fn iconParticleFlow(index: u32, phase: f32, flow: f32) -> vec2f {
  let edge = index % 8u;
  let along = fract(f32(index) * 0.6180339887 + phase * 0.01 + flow);
  let base = iconEdgePoint(edge, along);

  let nSeed = f32(index) * 0.017;
  let across = (hash11(nSeed) - 0.5) * FRAME_THICK * 0.85;
  let a = frameCorner(edge / 4u, edge % 4u);
  let b = frameCorner(edge / 4u, (edge % 4u + 1u) % 4u);
  let tangent = normalize(b - a);
  let normal = vec2f(-tangent.y, tangent.x);
  return base + normal * across;
}

export fn routePoint(routeId: u32, t: f32) -> vec2f {
  return iconEdgePoint(routeId % 8u, t);
}
