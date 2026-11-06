import type { ReactNode } from "react";

/** Text crossed by a moving glint (adapted from React Bits ShinyText; the sweep is CSS). */
export function ShinyText({ children }: { children: ReactNode }) {
  return <span className="shiny">{children}</span>;
}
