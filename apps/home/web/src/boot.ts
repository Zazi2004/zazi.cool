// Bootstrap: reads data-background and, if the visitor allows animation, loads the
// module of the chosen background once the content is on screen. Everything else
// (shaders, loop) lives in the module, so that visitors who do not animate download
// nothing more. No inline script: the CSP only allows this file (NF5).

import { animationAllowed, backgroundOf, currentEnv, type BackgroundName } from "./policy.ts";
import type { Start } from "./backgrounds/gl.ts";

const modules: Record<BackgroundName, () => Promise<{ default: Start }>> = {
  aurore: () => import("./backgrounds/aurore.ts"),
  fils: () => import("./backgrounds/fils.ts"),
  grain: () => import("./backgrounds/grain.ts"),
};

/** Calls fn when the browser is idle (or soon, where requestIdleCallback is missing). */
function whenIdle(fn: () => void): void {
  if ("requestIdleCallback" in window) window.requestIdleCallback(fn, { timeout: 1500 });
  else setTimeout(fn, 200);
}

async function run(name: BackgroundName, backdrop: Element): Promise<void> {
  const { default: start } = await modules[name]();
  const canvas = document.createElement("canvas");
  backdrop.append(canvas);
  // Read a style so the canvas is laid out at opacity 0 first: the opacity set by the
  // module is then a CSS transition (a fade-in over the static gradient).
  getComputedStyle(canvas).opacity;
  const dark = matchMedia("(prefers-color-scheme: dark)");
  const background = start(canvas, dark.matches ? "dark" : "light");
  if (!background) {
    canvas.remove();
    return;
  }
  dark.addEventListener("change", () => background.setScheme(dark.matches ? "dark" : "light"));
}

const name = backgroundOf(document.documentElement.dataset.background);
const backdrop = document.querySelector(".backdrop");
if (name && backdrop && animationAllowed(currentEnv())) {
  whenIdle(() => {
    run(name, backdrop).catch((err: unknown) => console.error("animated background failed", err));
  });
}
