struct Particle {
  pos: vec2f,
  vel: vec2f,
  phase: f32,
  route: f32,
  _pad: vec2f,
}

struct DrawParams {
  resolution: vec2f,
  particleSize: f32,
  brightness: f32,
}

@group(0) @binding(0) var<uniform> params: DrawParams;
@group(0) @binding(1) var<storage, read> particles: array<Particle>;

struct VertexOut {
  @builtin(position) position: vec4f,
  @location(0) coord: vec2f,
  @location(1) strength: f32,
}

fn quadCorner(vertexIndex: u32) -> vec2f {
  let i = vertexIndex % 6u;
  switch i {
    case 0u, 3u: { return vec2f(-1.0, -1.0); }
    case 1u, 4u: { return vec2f( 1.0, -1.0); }
    case 2u: { return vec2f(-1.0,  1.0); }
    default: { return vec2f( 1.0,  1.0); }
  }
}

@vertex fn vs_main(
  @builtin(vertex_index) vertexIndex: u32,
  @builtin(instance_index) instanceIndex: u32,
) -> VertexOut {
  let p = particles[instanceIndex];
  let aspect = params.resolution.x / max(params.resolution.y, 1.0);
  var pos = p.pos / 0.85;
  pos.x /= aspect;

  let corner = quadCorner(vertexIndex);
  let size = params.particleSize / params.resolution.y;
  let clip = pos + corner * size;

  var out: VertexOut;
  out.position = vec4f(clip, 0.0, 1.0);
  out.coord = corner * 0.5 + 0.5;
  out.strength = params.brightness * (0.75 + min(length(p.vel) * 4.0, 0.5));
  return out;
}

@fragment fn fs_main(in: VertexOut) -> @location(0) vec4f {
  let d = length(in.coord - vec2f(0.5));
  if (d > 0.5) { discard; }
  let core = 1.0 - smoothstep(0.0, 0.22, d);
  let halo = 1.0 - smoothstep(0.15, 0.5, d);
  let falloff = core * 0.85 + halo * 0.45;
  let color = vec3f(0.05, 0.98, 0.92) * in.strength * falloff;
  let alpha = falloff * in.strength * 0.55;
  return vec4f(color, alpha);
}
