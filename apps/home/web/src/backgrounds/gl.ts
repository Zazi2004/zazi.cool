// WebGL 2 plumbing shared by the three backgrounds: shader programs, a render size
// capped like the linktree's, and the animation loop.

import type { Scheme } from "../policy.ts";

/** A running background; the module sets the canvas opacity of the scheme it draws. */
export interface Background {
  setScheme(scheme: Scheme): void;
}

/** Starts a background on a canvas, or returns null if WebGL 2 is not usable. */
export type Start = (canvas: HTMLCanvasElement, scheme: Scheme) => Background | null;

export type Uniforms = (name: string) => WebGLUniformLocation | null;

const MAX_RATIO = 1.5;
const MAX_DIMENSION = 1920;

/** Pixels to render for a CSS size: the device ratio is capped, and so is the longest side. */
export function renderSize(
  width: number,
  height: number,
  deviceRatio: number,
): { width: number; height: number } {
  let ratio = Math.min(deviceRatio || 1, MAX_RATIO);
  const longest = Math.max(width, height) * ratio;
  if (longest > MAX_DIMENSION) ratio = (ratio * MAX_DIMENSION) / longest;
  return {
    width: Math.max(1, Math.round(width * ratio)),
    height: Math.max(1, Math.round(height * ratio)),
  };
}

function compile(gl: WebGL2RenderingContext, type: number, source: string): WebGLShader {
  const shader = gl.createShader(type);
  if (!shader) throw new Error("cannot create a shader");
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    throw new Error(`shader: ${gl.getShaderInfoLog(shader)}`);
  }
  return shader;
}

/** Compiles and links a program; the attributes get the locations 0, 1… in order. */
export function createProgram(
  gl: WebGL2RenderingContext,
  vertex: string,
  fragment: string,
  attributes: string[] = [],
): { program: WebGLProgram; uniform: Uniforms } {
  const program = gl.createProgram();
  if (!program) throw new Error("cannot create a program");
  const shaders = [
    compile(gl, gl.VERTEX_SHADER, vertex),
    compile(gl, gl.FRAGMENT_SHADER, fragment),
  ];
  for (const shader of shaders) gl.attachShader(program, shader);
  attributes.forEach((name, i) => gl.bindAttribLocation(program, i, name));
  gl.linkProgram(program);
  for (const shader of shaders) gl.deleteShader(shader);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    throw new Error(`program: ${gl.getProgramInfoLog(program)}`);
  }
  // An unused uniform has no location: setting it is then silently ignored by WebGL.
  return { program, uniform: (name) => gl.getUniformLocation(program, name) };
}

export function createTexture(gl: WebGL2RenderingContext, filter: number): WebGLTexture {
  const texture = gl.createTexture();
  gl.bindTexture(gl.TEXTURE_2D, texture);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
  return texture;
}

/** A texture of the given size, with a framebuffer drawing into it. */
export function resizeTarget(
  gl: WebGL2RenderingContext,
  texture: WebGLTexture,
  framebuffer: WebGLFramebuffer,
  width: number,
  height: number,
): void {
  gl.bindTexture(gl.TEXTURE_2D, texture);
  gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, null);
  gl.bindFramebuffer(gl.FRAMEBUFFER, framebuffer);
  gl.framebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture, 0);
  gl.bindFramebuffer(gl.FRAMEBUFFER, null);
}

export function getContext(
  canvas: HTMLCanvasElement,
  alpha: boolean,
): WebGL2RenderingContext | null {
  return canvas.getContext("webgl2", {
    alpha,
    premultipliedAlpha: true,
    antialias: false,
    depth: false,
    stencil: false,
  });
}

/**
 * Runs the animation: onSize is called when the canvas size changes (the canvas is
 * resized first), onFrame(dt) once per frame with the seconds since the last one
 * (0 for a redraw after a resize). The loop pauses while the tab is hidden (F23).
 */
export function animate(
  canvas: HTMLCanvasElement,
  onSize: (width: number, height: number) => void,
  onFrame: (dt: number) => void,
): void {
  let raf = 0;
  let last = 0;

  const tick = (now: number) => {
    raf = 0;
    if (document.hidden) return;
    // A long pause (hidden tab, slow frame) must not make the animation jump.
    const dt = last ? Math.min(0.05, (now - last) / 1000) : 1 / 60;
    last = now;
    onFrame(dt);
    raf = requestAnimationFrame(tick);
  };

  const onVisibility = () => {
    if (document.hidden) {
      cancelAnimationFrame(raf);
      raf = 0;
      last = 0;
    } else if (!raf) {
      raf = requestAnimationFrame(tick);
    }
  };

  const resize = () => {
    const size = renderSize(canvas.clientWidth, canvas.clientHeight, devicePixelRatio);
    canvas.width = size.width;
    canvas.height = size.height;
    onSize(size.width, size.height);
    onFrame(0);
  };

  document.addEventListener("visibilitychange", onVisibility);
  new ResizeObserver(resize).observe(canvas);
  resize();
  onVisibility();
}
