import { nexlayerGlow } from "./nexlayer-sdf.wgsl";

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
@group(0) @binding(1) var particleTex: texture_2d<f32>;
@group(0) @binding(2) var particleSampler: sampler;

fn uvToLogoSpace(uv: vec2f) -> vec2f {
  let aspect = params.resolution.x / max(params.resolution.y, 1.0);
  var p = uv * 2.0 - 1.0;
  p.x *= aspect;
  return p * 0.85;
}

@fragment fn fs_main(@location(0) uv: vec2f) -> @location(0) vec4f {
  let p = uvToLogoSpace(uv);
  let time = params.time;

  let radial = length(p);
  var bg = vec3f(0.015, 0.025, 0.04);
  bg += vec3f(0.0, 0.04, 0.045) * exp(-radial * 1.8) * 0.35;

  let underpaint = nexlayerGlow(p, 0.25 * params.edgeSharpness);
  let cyan = vec3f(0.0, 0.95, 0.88);

  let swarm = textureSampleLevel(particleTex, particleSampler, uv, 0.0).rgb;

  var ghost = 0.0;
  if (params.state > 3.5 && params.state < 4.5) {
    for (var i = 0; i < 3; i++) {
      let angle = f32(i) * 2.094 + time * 0.2;
      let offset = vec2f(cos(angle), sin(angle)) * 0.28 * sin(params.stateProgress * 3.14159);
      ghost = max(ghost, nexlayerGlow(p - offset, 0.15));
    }
  }

  var shock = 0.0;
  if (params.shockwave > 0.0 && params.shockwave < 1.0) {
    let ring = abs(radial - params.shockwave * 0.9);
    shock = exp(-ring * 45.0) * (1.0 - params.shockwave) * 0.7;
  }

  var pointerGlow = 0.0;
  if (params.pointerActive > 0.5) {
    let pp = uvToLogoSpace(params.pointer);
    pointerGlow = exp(-length(p - pp) * 5.0) * 0.1;
  }

  let breathe = sin(time * 1.4) * 0.5 + 0.5;
  let pulse = 0.9 + breathe * 0.1;

  var color = bg;
  color += cyan * underpaint * 0.35 * pulse;
  color += cyan * ghost * 0.4;
  color += swarm * 1.35;
  color += cyan * shock;
  color += vec3f(0.0, 0.85, 0.8) * pointerGlow;

  let vig = 1.0 - dot((uv - 0.5) * 1.15, (uv - 0.5) * 1.15);
  color *= clamp(vig, 0.45, 1.0);

  return vec4f(color, 1.0);
}
