// "fils": luminous threads in the accent violet and cyan (the "neon" background of the
// linktree). Shaders: fils.shaders.ts. The threads are drawn into a texture, then
// brightened and composited onto the canvas.

import palettes from "../backgrounds.json";
import type { Scheme } from "../policy.ts";
import { hexToRgb } from "./color.ts";
import {
  animate,
  createProgram,
  createTexture,
  getContext,
  resizeTarget,
  type Start,
} from "./gl.ts";
import { LINE_FRAGMENT, LINE_VERTEX, SCREEN_FRAGMENT, SCREEN_VERTEX } from "./fils.shaders.ts";

const SPEED = 0.5;
const LINES = 90;
const ANGLE = (25 * Math.PI) / 180;
const BRIGHTNESS = 1.4;

const start: Start = (canvas, scheme) => {
  const gl = getContext(canvas, true);
  if (!gl) return null;
  const lines = createProgram(gl, LINE_VERTEX, LINE_FRAGMENT, ["aPoint"]);
  const screen = createProgram(gl, SCREEN_VERTEX, SCREEN_FRAGMENT, ["aPosition"]);

  // Each thread is a triangle strip of `segments` quads, repeated by instancing.
  const lineVao = gl.createVertexArray();
  const lineBuffer = gl.createBuffer();
  gl.bindVertexArray(lineVao);
  gl.bindBuffer(gl.ARRAY_BUFFER, lineBuffer);
  gl.enableVertexAttribArray(0);
  gl.vertexAttribPointer(0, 2, gl.FLOAT, false, 0, 0);

  const screenVao = gl.createVertexArray();
  gl.bindVertexArray(screenVao);
  gl.bindBuffer(gl.ARRAY_BUFFER, gl.createBuffer());
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
  gl.enableVertexAttribArray(0);
  gl.vertexAttribPointer(0, 2, gl.FLOAT, false, 0, 0);

  const texture = createTexture(gl, gl.NEAREST);
  const framebuffer = gl.createFramebuffer();

  let segments = 0;
  let color = hexToRgb("#ffffff");
  let accent = color;
  let time = 0;

  const setScheme = (next: Scheme) => {
    const { colors, opacity } = palettes.fils[next];
    color = hexToRgb(colors[0] ?? "#ffffff");
    accent = hexToRgb(colors[1] ?? "#ffffff");
    canvas.style.opacity = String(opacity);
  };
  setScheme(scheme);

  const buildStrip = (count: number) => {
    if (count === segments) return;
    segments = count;
    const points = new Float32Array((count + 1) * 4);
    for (let i = 0; i <= count; i++) points.set([i / count, -1, i / count, 1], i * 4);
    gl.bindBuffer(gl.ARRAY_BUFFER, lineBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, points, gl.STATIC_DRAW);
  };

  animate(
    canvas,
    (width, height) => {
      resizeTarget(gl, texture, framebuffer, width, height);
      buildStrip(Math.min(640, Math.max(96, Math.round(canvas.clientWidth / 3))));
      gl.viewport(0, 0, width, height);
    },
    (dt) => {
      time += dt * SPEED;
      const w = canvas.width;
      const h = canvas.height;
      const cos = Math.cos(ANGLE);
      const sin = Math.sin(ANGLE);
      const u = lines.uniform;

      gl.bindFramebuffer(gl.FRAMEBUFFER, framebuffer);
      gl.clearColor(0, 0, 0, 0);
      gl.clear(gl.COLOR_BUFFER_BIT);
      gl.enable(gl.BLEND);
      gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA);
      gl.useProgram(lines.program);
      gl.uniform1f(u("uTime"), time);
      gl.uniform2f(u("uResolution"), w, h);
      gl.uniform2f(
        u("uFrame"),
        Math.abs(w * cos) + Math.abs(h * sin),
        Math.abs(w * sin) + Math.abs(h * cos),
      );
      gl.uniform1f(u("uAmplitude"), 1);
      gl.uniform1f(u("uDistance"), 0.2);
      gl.uniform2f(u("uMouse"), 0.5, 0.5);
      gl.uniform1f(u("uCount"), LINES);
      gl.uniform1f(u("uThickness"), 0.6);
      gl.uniform1f(u("uSoftness"), 1.3);
      gl.uniform1f(u("uWaves"), 1.05);
      gl.uniform1f(u("uSplit"), 0.04);
      gl.uniform1f(u("uFray"), 0.5);
      gl.uniform1f(u("uSeed"), 0);
      gl.uniform2f(u("uTurn"), cos, sin);
      gl.uniform4f(u("uPart"), 0, 0, 0, Math.min(w, h) * 0.16);
      gl.uniform1f(u("uTaper"), 0.85);
      gl.uniform3f(u("uColor"), ...color);
      gl.uniform3f(u("uAccent"), ...accent);
      gl.bindVertexArray(lineVao);
      gl.drawArraysInstanced(gl.TRIANGLE_STRIP, 0, (segments + 1) * 2, LINES);

      gl.bindFramebuffer(gl.FRAMEBUFFER, null);
      gl.disable(gl.BLEND);
      gl.clear(gl.COLOR_BUFFER_BIT);
      gl.useProgram(screen.program);
      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, texture);
      gl.uniform1i(screen.uniform("uLines"), 0);
      gl.uniform2f(screen.uniform("uResolution"), w, h);
      gl.uniform1f(screen.uniform("uFade"), 0);
      gl.uniform1f(screen.uniform("uOpacity"), 1);
      gl.uniform1f(screen.uniform("uCurve"), 2 / BRIGHTNESS);
      gl.bindVertexArray(screenVao);
      gl.drawArrays(gl.TRIANGLES, 0, 3);
    },
  );
  return { setScheme };
};

export default start;
