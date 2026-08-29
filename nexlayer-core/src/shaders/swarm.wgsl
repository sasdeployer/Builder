import {
  hash11,
  iconEdgePoint,
  iconParticleFlow,
  iconParticleHome,
  nexlayerGlow,
  nexlayerSdf,
} from "./nexlayer-sdf.wgsl";

struct RenderParams {
  time: f32,
  _pad0: f32,
  resolution: vec2f,
  state: f32,
  stateProgress: f32,
  pointer: vec2f,
  pointerActive: f32,
  edgeSharpness: f32,
  shockwave: f32,
  scaleCopies: f32,
}

@group(0) @binding(0) var<uniform> params: RenderParams;

const SWARM_COUNT: u32 = 320u;

fn uvToLogoSpace(uv: vec2f) -> vec2f {
  let aspect = params.resolution.x / max(params.resolution.y, 1.0);
  var p = uv * 2.0 - 1.0;
  p.x *= aspect;
  return p * 0.85;
}

fn pointerWorld(uv: vec2f) -> vec2f {
  return uvToLogoSpace(uv);
}

fn particleHome(i: u32) -> vec2f {
  let phase = hash11(f32(i) * 2.71);
  let state = params.state;
  let flow = params.time * (0.035 + hash11(f32(i) * 0.91) * 0.02);

  // Idle / default: particles live on the stroke — swarm creates the mark
  var home = iconParticleFlow(i, phase, flow * 0.5);

  if (state > 0.5 && state < 1.5) {
    home = iconParticleFlow(i, phase, flow * 1.05);
    if (params.pointerActive > 0.5) {
      let pw = pointerWorld(params.pointer);
      let delta = home - pw;
      let dist = length(delta);
      if (dist < 0.28) {
        home += normalize(delta + vec2f(0.001)) * (0.28 - dist) * 0.55;
      }
    }
  }

  if (state > 1.5 && state < 2.5) {
    let t = params.stateProgress;
    if (t < 0.32) {
      let angle = f32(i) * 2.399963 + params.time * 2.0;
      let radius = 0.12 + hash11(f32(i)) * 0.5 * (t / 0.32);
      home = vec2f(cos(angle), sin(angle)) * radius;
    } else if (t < 0.62) {
      let sub = u32(hash11(f32(i) * 0.53 + floor(params.time * 3.0)) * 8.0);
      home = iconEdgePoint(sub, fract(params.time * 0.4 + f32(i) * 0.001));
    } else {
      let snap = smoothstep(0.0, 1.0, (t - 0.62) / 0.38);
      let assembled = iconParticleHome(i, phase);
      let angle = f32(i) * 2.399963;
      let scattered = vec2f(cos(angle), sin(angle)) * 0.35;
      home = mix(scattered, assembled, snap);
    }
  }

  if (state > 2.5 && state < 3.5) {
    let t = params.stateProgress;
    let lane = f32(i % 8u) / 8.0;
    if (t < 0.22) {
      home = mix(vec2f(-0.7, lane * 0.55 - 0.27), iconEdgePoint(i % 8u, 0.0), t / 0.22);
    } else if (t < 0.72) {
      home = iconParticleFlow(i, phase, (t - 0.22) / 0.5);
    } else {
      home = mix(iconEdgePoint(i % 8u, 1.0), vec2f(0.72, 0.0), (t - 0.72) / 0.28);
    }
  }

  if (state > 3.5 && state < 4.5) {
    let copy = i % 4u;
    let strength = sin(params.stateProgress * 3.14159);
    let angle = f32(copy) * 1.5708 + params.time * 0.25;
    let offset = vec2f(cos(angle), sin(angle)) * 0.3 * strength;
    let scale = 1.0 - strength * 0.22 * select(0.0, 1.0, copy > 0u);
    home = iconParticleFlow(i, phase, flow) * scale + offset;
  }

  if (state > 4.5 && state < 5.5) {
    home = iconParticleHome(i, phase);
  }

  if (state > 5.5) {
    let chaosAmt = (1.0 - params.stateProgress);
    if (chaosAmt > 0.08) {
      home = iconParticleFlow(i, phase, flow * 2.2);
      home += vec2f(
        sin(params.time * 14.0 + f32(i)),
        cos(params.time * 11.0 + f32(i) * 1.3)
      ) * chaosAmt * 0.28;
    } else {
      home = iconParticleHome(i, phase);
    }
  }

  return home;
}

fn swarmField(p: vec2f) -> f32 {
  if (length(p) > 0.85) { return 0.0; }

  var energy = 0.0;
  for (var i = 0u; i < SWARM_COUNT; i++) {
    let pos = particleHome(i);
    let d = length(p - pos);
    let size = 0.018 + hash11(f32(i) * 1.7) * 0.012;
    energy += exp(-d * d / (size * size)) * (0.6 + hash11(f32(i) * 0.3) * 0.4);
  }
  return energy;
}

@fragment fn fs_main(@location(0) uv: vec2f) -> @location(0) vec4f {
  let p = uvToLogoSpace(uv);
  let time = params.time;
  let radial = length(p);

  var bg = vec3f(0.012, 0.02, 0.035);
  bg += vec3f(0.0, 0.035, 0.04) * exp(-radial * 1.6) * 0.4;

  let cyan = vec3f(0.0, 0.96, 0.9);

  let under = nexlayerGlow(p, 0.18 * params.edgeSharpness);
  let swarm = swarmField(p);

  let inside = smoothstep(0.04, -0.12, nexlayerSdf(p));
  let coreGlow = inside * 0.04 * (0.6 + 0.4 * sin(time * 1.2));

  var ghost = 0.0;
  if (params.state > 3.5 && params.state < 4.5) {
    for (var g = 0; g < 3; g++) {
      let angle = f32(g) * 2.094 + time * 0.2;
      let offset = vec2f(cos(angle), sin(angle)) * 0.3 * sin(params.stateProgress * 3.14159);
      ghost = max(ghost, nexlayerGlow(p - offset, 0.12));
    }
  }

  var shock = 0.0;
  if (params.shockwave > 0.0 && params.shockwave < 1.0) {
    let ring = abs(radial - params.shockwave * 0.9);
    shock = exp(-ring * 45.0) * (1.0 - params.shockwave) * 0.75;
  }

  var pointerGlow = 0.0;
  if (params.pointerActive > 0.5) {
    let pp = pointerWorld(params.pointer);
    pointerGlow = exp(-length(p - pp) * 5.0) * 0.12;
  }

  let breathe = sin(time * 1.35) * 0.5 + 0.5;
  let pulse = 0.88 + breathe * 0.12;

  let mark = 1.0 - exp(-swarm * 0.12);
  let markBright = pow(mark, 0.85) * pulse;

  var color = bg;
  color += cyan * under * 0.22;
  color += cyan * coreGlow;
  color += cyan * ghost * 0.35;
  color += cyan * markBright;
  color += vec3f(0.55, 1.0, 0.98) * pow(mark, 2.2) * 0.45;
  color += cyan * shock;
  color += vec3f(0.0, 0.85, 0.8) * pointerGlow;

  let vig = 1.0 - dot((uv - 0.5) * 1.15, (uv - 0.5) * 1.15);
  color *= clamp(vig, 0.45, 1.0);

  return vec4f(color, 1.0);
}
