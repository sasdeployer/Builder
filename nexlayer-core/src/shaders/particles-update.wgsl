import {
  hash11,
  iconParticleFlow,
  iconParticleHome,
  iconEdgePoint,
} from "./nexlayer-sdf.wgsl";

const PARTICLE_COUNT: u32 = 4096u;
const WORKGROUP_SIZE: u32 = 256u;

struct SimParams {
  time: f32,
  dt: f32,
  state: f32,
  stateProgress: f32,
  pointer: vec2f,
  pointerActive: f32,
  launchPhase: f32,
  shockwave: f32,
  chaos: f32,
  scaleCopies: f32,
  _pad0: f32,
  _pad1: vec2f,
}

struct Particle {
  pos: vec2f,
  vel: vec2f,
  phase: f32,
  route: f32,
  _pad: vec2f,
}

@group(0) @binding(0) var<uniform> params: SimParams;
@group(0) @binding(1) var<storage, read_write> particles: array<Particle>;

fn magneticField(p: vec2f, pointer: vec2f, isActive: f32, strength: f32) -> vec2f {
  let delta = pointer - p;
  let dist = length(delta) + 0.001;
  let falloff = exp(-dist * 4.0) * isActive * strength;
  return normalize(delta) * falloff;
}

fn pointerWorld(uv: vec2f) -> vec2f {
  var p = uv * 2.0 - 1.0;
  return p * 0.85;
}

@compute @workgroup_size(WORKGROUP_SIZE)
fn cs_init(@builtin(global_invocation_id) gid: vec3u) {
  let i = gid.x;
  if (i >= PARTICLE_COUNT) { return; }

  let seed = f32(i);
  let phase = hash11(seed * 2.71);
  let home = iconParticleHome(i, phase);
  let scatter = (vec2f(hash11(seed * 3.1), hash11(seed * 4.7)) - 0.5) * 0.55;

  var particle: Particle;
  particle.pos = home + scatter;
  particle.vel = vec2f(0.0);
  particle.phase = phase;
  particle.route = f32(i % 8u);
  particle._pad = vec2f(0.0);
  particles[i] = particle;
}

@compute @workgroup_size(WORKGROUP_SIZE)
fn cs_update(@builtin(global_invocation_id) gid: vec3u) {
  let i = gid.x;
  if (i >= PARTICLE_COUNT) { return; }

  var p = particles[i];
  let state = params.state;
  let dt = params.dt;

  var flowSpeed = 0.04 + hash11(f32(i) * 0.91) * 0.025;
  var routeTarget = iconParticleFlow(i, p.phase, params.time * flowSpeed);
  var spring = 10.0;

  if (state < 0.5) {
    let breathe = sin(params.time * 1.1 + f32(i) * 0.008) * 0.004;
    routeTarget = iconParticleFlow(i, p.phase, params.time * flowSpeed * 0.45);
    routeTarget += vec2f(breathe, breathe * 0.4);
    spring = 14.0;
  }

  if (state > 0.5 && state < 1.5) {
    routeTarget = iconParticleFlow(i, p.phase, params.time * flowSpeed * 1.1);
    if (params.pointerActive > 0.5) {
      let pw = pointerWorld(params.pointer);
      let field = magneticField(p.pos, pw, 1.0, 0.18);
      p.vel += field * dt * 4.0;
      let dist = length(p.pos - pw);
      if (dist < 0.22) {
        let push = normalize(p.pos - pw + vec2f(0.001)) * (0.22 - dist) * 0.35;
        routeTarget += push;
      }
    }
    spring = 9.0;
  }

  if (state > 1.5 && state < 2.5) {
    let t = params.launchPhase;
    if (t < 0.32) {
      let angle = f32(i) * 2.399963 + params.time;
      routeTarget = vec2f(cos(angle), sin(angle)) * (0.15 + hash11(f32(i)) * 0.4);
      spring = 3.0;
    } else if (t < 0.62) {
      let sub = u32(hash11(f32(i) * 0.53 + floor(params.time * 2.0)) * 8.0);
      routeTarget = iconEdgePoint(sub, fract(params.time * 0.35 + f32(i) * 0.001));
      spring = 5.0;
    } else {
      let snap = smoothstep(0.0, 1.0, (t - 0.62) / 0.38);
      let home = iconParticleHome(i, p.phase);
      routeTarget = mix(p.pos, home, snap);
      spring = 6.0 + snap * 16.0;
    }
  }

  if (state > 2.5 && state < 3.5) {
    let t = params.stateProgress;
    let lane = f32(i % 8u) / 8.0;
    if (t < 0.22) {
      routeTarget = mix(
        vec2f(-0.65, lane * 0.5 - 0.25),
        iconEdgePoint(i % 8u, 0.0),
        t / 0.22
      );
      spring = 8.0;
    } else if (t < 0.72) {
      routeTarget = iconParticleFlow(i, p.phase, (t - 0.22) / 0.5);
      spring = 11.0;
    } else {
      let exitT = (t - 0.72) / 0.28;
      routeTarget = mix(iconEdgePoint(i % 8u, 1.0), vec2f(0.7, 0.0), exitT);
      spring = 7.0;
    }
  }

  if (state > 3.5 && state < 4.5) {
    let copy = i % 4u;
    let strength = sin(params.stateProgress * 3.14159);
    let angle = f32(copy) * 1.5708 + params.time * 0.25;
    let offset = vec2f(cos(angle), sin(angle)) * 0.28 * strength;
    let scale = 1.0 - strength * 0.25 * select(0.0, 1.0, copy > 0u);
    routeTarget = iconParticleFlow(i, p.phase, params.time * flowSpeed) * scale + offset;
    spring = 10.0;
  }

  if (state > 4.5 && state < 5.5) {
    routeTarget = iconParticleHome(i, p.phase);
    spring = 22.0;
    let wave = params.shockwave;
    if (wave > 0.0 && wave < 1.0) {
      let dist = length(p.pos);
      let ring = abs(dist - wave * 0.85);
      p.vel += normalize(p.pos + vec2f(0.001)) * exp(-ring * 35.0) * 0.55 * (1.0 - wave);
    }
  }

  if (state > 5.5) {
    let chaosAmt = params.chaos * (1.0 - params.stateProgress);
    if (chaosAmt > 0.05) {
      routeTarget = iconParticleFlow(i, p.phase, params.time * flowSpeed * 2.0);
      routeTarget += vec2f(
        sin(params.time * 14.0 + f32(i)),
        cos(params.time * 11.0 + f32(i) * 1.3)
      ) * chaosAmt * 0.3;
      if (hash11(f32(i) + floor(params.time * 4.0)) > 0.65) {
        routeTarget = iconEdgePoint(u32(hash11(f32(i) * 3.1) * 8.0), fract(params.time * 0.4));
      }
      spring = 4.0;
    } else {
      routeTarget = iconParticleHome(i, p.phase);
      spring = 16.0;
    }
  }

  let toTarget = routeTarget - p.pos;
  p.vel += toTarget * spring * dt;
  p.vel *= exp(-dt * 4.5);

  if (params.pointerActive > 0.5 && state < 1.5) {
    let pw = pointerWorld(params.pointer);
    p.vel += magneticField(p.pos, pw, 1.0, 0.06) * dt * 2.0;
  }

  p.pos += p.vel * dt;
  p.phase += flowSpeed * dt * 0.08;
  particles[i] = p;
}
