import { useRef, type PointerEvent, type ReactNode } from "react";

/** A card with a soft light following the pointer (adapted from React Bits SpotlightCard). */
export function SpotlightCard({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  const node = useRef<HTMLDivElement>(null);

  function move(e: PointerEvent) {
    const el = node.current;
    if (!el) return;
    const box = el.getBoundingClientRect();
    // Custom properties are set through the CSSOM, which the CSP allows.
    el.style.setProperty("--spot-x", `${e.clientX - box.left}px`);
    el.style.setProperty("--spot-y", `${e.clientY - box.top}px`);
  }

  return (
    <div ref={node} className={`spotlight ${className}`.trim()} onPointerMove={move}>
      {children}
    </div>
  );
}
