import { animate } from "motion";
import { useEffect, useRef } from "react";
import { prefersReducedMotion } from "./motion";

/** A number that counts up to `to` (adapted from React Bits CountUp, on Motion). */
export function CountUp({ to, suffix = "" }: { to: number; suffix?: string }) {
  const node = useRef<HTMLSpanElement>(null);
  const shown = useRef(0);

  useEffect(() => {
    const el = node.current;
    if (!el) return;
    const write = (n: number) => {
      shown.current = n;
      el.textContent = `${Math.round(n)}${suffix}`;
    };
    if (prefersReducedMotion()) {
      write(to);
      return;
    }
    const controls = animate(shown.current, to, {
      duration: 0.8,
      ease: "easeOut",
      onUpdate: write,
    });
    return () => controls.stop();
  }, [to, suffix]);

  return <span ref={node}>{`${to}${suffix}`}</span>;
}
