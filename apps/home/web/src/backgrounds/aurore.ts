// "aurore": an aurora in the accent violet and cyan (the "nuit" background of the
// linktree). Shader: aurore.shaders.ts.

import palettes from "../backgrounds.json";
import type { Scheme } from "../policy.ts";
import { hexToRgb, toOklch, unwrapHues } from "./color.ts";
import { animate, createProgram, getContext, type Start } from "./gl.ts";
import { FRAGMENT, MAX_STOPS, VERTEX } from "./aurore.shaders.ts";

const SPEED = 0.6;

const start: Start = (canvas, scheme) => {
  const gl = getContext(canvas, true);
  if (!gl) return null;
  const { program, uniform } = createProgram(gl, VERTEX, FRAGMENT, ["aPosition"]);

  // One oversized triangle covers the screen.
  const vao = gl.createVertexArray();
  gl.bindVertexArray(vao);
  gl.bindBuffer(gl.ARRAY_BUFFER, gl.createBuffer());
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
  gl.enableVertexAttribArray(0);
  gl.vertexAttribPointer(0, 2, gl.FLOAT, false, 0, 0);

  const stops = new Float32Array(MAX_STOPS * 3);
  let count = 0;
  let light = false;
  let time = 0;

  const setScheme = (next: Scheme) => {
    const { colors, opacity } = palettes.aurore[next];
    unwrapHues(colors.map((hex) => toOklch(hexToRgb(hex)))).forEach((stop, i) =>
      stops.set(stop, i * 3),
    );
    count = colors.length;
    light = next === "light";
    canvas.style.opacity = String(opacity);
  };
  setScheme(scheme);

  animate(
    canvas,
    (width, height) => gl.viewport(0, 0, width, height),
    (dt) => {
      time += dt * SPEED;
      gl.useProgram(program);
      gl.uniform2f(uniform("uResolution"), canvas.width, canvas.height);
      gl.uniform1f(uniform("uTime"), time);
      gl.uniform3fv(uniform("uStops"), stops);
      gl.uniform1i(uniform("uStopCount"), count);
      gl.uniform1f(uniform("uAmplitude"), 1.1);
      gl.uniform1f(uniform("uBlend"), 0.6);
      gl.uniform1f(uniform("uHeight"), 0.55);
      gl.uniform1f(uniform("uScale"), 1);
      gl.uniform1f(uniform("uGlow"), 0.6);
      gl.uniform1f(uniform("uBrightness"), 1);
      gl.uniform1i(uniform("uLayers"), 1);
      gl.uniform1f(uniform("uFlip"), 0);
      gl.uniform3f(uniform("uPointer"), 0, 0.5, 0);
      gl.uniform1f(uniform("uReveal"), 1);
      // The canvas opacity (backgrounds.json) is applied by CSS, not by the shader.
      gl.uniform1f(uniform("uOpacity"), 1);
      gl.uniform1f(uniform("uLightMode"), light ? 1 : 0);
      gl.drawArrays(gl.TRIANGLES, 0, 3);
    },
  );
  return { setScheme };
};

export default start;
