/** Whether the user asked for less motion; read when an effect starts, not cached. */
export function prefersReducedMotion(): boolean {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}
