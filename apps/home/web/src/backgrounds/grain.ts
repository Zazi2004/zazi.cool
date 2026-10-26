// "grain": a slowly moving grainy gradient (the "pop" background of the linktree).
// Shaders: grain.shaders.ts. The gradient is drawn at half resolution into a texture,
// then the grain is added at full resolution.

import palettes from "../backgrounds.json";
import type { Scheme } from "../policy.ts";
import { hexToRgb, toOklab } from "./color.ts";
import {
  animate,
  createProgram,
  createTexture,
  getContext,
  resizeTarget,
  type Start,
} from "./gl.ts";
import { COMPOSITE, FIELD, MAX_COLORS, VERTEX } from "./grain.shaders.ts";

const SPEED = 0.8 * 0.16;
const ANGLE = (55 * Math.PI) / 180;
const GRAIN_STEP = 1 / 24;

const start: Start = (canvas, scheme) => {
  const gl = getContext(canvas, false);
  if (!gl) return null;
  const field = createProgram(gl, VERTEX, FIELD);
  const composite = createProgram(gl, VERTEX, COMPOSITE);
  // The vertex shader draws a triangle from gl_VertexID: it only needs a bound VAO.
  gl.bindVertexArray(gl.createVertexArray());
  const texture = createTexture(gl, gl.LINEAR);
  const framebuffer = gl.createFramebuffer();
  const fieldSize = { width: 1, height: 1 };

  const ramp = new Float32Array(MAX_COLORS * 3);
  let count = 0;
  let time = 0;
  let grainClock = 0;
  let grainFrame = 0;

  const setScheme = (next: Scheme) => {
    const { colors, opacity } = palettes.grain[next];
    colors.forEach((hex, i) => ramp.set(toOklab(hexToRgb(hex)), i * 3));
    count = colors.length;
    canvas.style.opacity = String(opacity);
  };
  setScheme(scheme);

  animate(
    canvas,
    (width, height) => {
      fieldSize.width = Math.max(1, Math.round(width / 2));
      fieldSize.height = Math.max(1, Math.round(height / 2));
      resizeTarget(gl, texture, framebuffer, fieldSize.width, fieldSize.height);
    },
    (dt) => {
      time += dt * SPEED;
      grainClock += dt;
      if (grainClock > GRAIN_STEP) {
        grainClock = 0;
        grainFrame = (grainFrame + 1) % 97;
      }
      const aspect = canvas.width / canvas.height;
      const u = field.uniform;

      gl.bindFramebuffer(gl.FRAMEBUFFER, framebuffer);
      gl.viewport(0, 0, fieldSize.width, fieldSize.height);
      gl.useProgram(field.program);
      gl.uniform1f(u("uAspect"), aspect);
      gl.uniform1f(u("uTime"), time);
      gl.uniform3fv(u("uRamp"), ramp);
      gl.uniform1i(u("uCount"), count);
      gl.uniform1i(u("uType"), 0);
      gl.uniform2f(u("uDir"), Math.sin(ANGLE), Math.cos(ANGLE));
      gl.uniform1f(u("uExtent"), aspect * Math.abs(Math.sin(ANGLE)) + Math.abs(Math.cos(ANGLE)));
      gl.uniform1f(u("uBands"), 2);
      gl.uniform1f(u("uSoftness"), 1);
      gl.uniform1f(u("uFlow"), 1);
      gl.uniform1f(u("uScale"), 1);
      gl.uniform1f(u("uSilk"), 0.35);
      gl.uniform2f(u("uSeed"), 0, 0);
      gl.uniform1f(u("uContrast"), 1);
      gl.uniform1f(u("uSaturation"), 1);
      gl.uniform1f(u("uSettle"), 1);
      gl.uniform1f(u("uReveal"), 1);
      gl.drawArrays(gl.TRIANGLES, 0, 3);

      gl.bindFramebuffer(gl.FRAMEBUFFER, null);
      gl.viewport(0, 0, canvas.width, canvas.height);
      gl.useProgram(composite.program);
      gl.activeTexture(gl.TEXTURE0);
      gl.bindTexture(gl.TEXTURE_2D, texture);
      gl.uniform1i(composite.uniform("uField"), 0);
      gl.uniform1f(composite.uniform("uGrain"), 0.16);
      gl.uniform1f(composite.uniform("uGrainSize"), 1.2 * (canvas.width / canvas.clientWidth));
      gl.uniform1f(composite.uniform("uFrame"), grainFrame);
      gl.drawArrays(gl.TRIANGLES, 0, 3);
    },
  );
  return { setScheme };
};

export default start;
