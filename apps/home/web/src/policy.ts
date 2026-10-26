// When a background may run (F23) and which one the page asks for (F22).

export const backgroundNames = ["aurore", "fils", "grain"] as const;
export type BackgroundName = (typeof backgroundNames)[number];

export type Scheme = "light" | "dark";

export interface Env {
  webgl2: boolean;
  reducedMotion: boolean;
  saveData: boolean;
}

/**
 * Whether an animated background may run: WebGL 2 is available, and the visitor
 * asked neither for less motion nor to save data. Otherwise the static gradient
 * stays alone and no module is downloaded.
 */
export function animationAllowed(env: Env): boolean {
  return env.webgl2 && !env.reducedMotion && !env.saveData;
}

/** The animated background named by data-background; "statique" and unknown names give null. */
export function backgroundOf(value: string | undefined): BackgroundName | null {
  return backgroundNames.find((name) => name === value) ?? null;
}

/** Whether a WebGL 2 context can really be created (the API can exist with the GPU disabled). */
function hasWebGL2(): boolean {
  if (typeof WebGL2RenderingContext === "undefined") return false;
  const gl = document.createElement("canvas").getContext("webgl2");
  gl?.getExtension("WEBGL_lose_context")?.loseContext();
  return gl !== null;
}

export function currentEnv(): Env {
  const connection = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection;
  return {
    webgl2: hasWebGL2(),
    reducedMotion: matchMedia("(prefers-reduced-motion: reduce)").matches,
    saveData: connection?.saveData === true,
  };
}
