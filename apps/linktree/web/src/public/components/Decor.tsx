import { m, useReducedMotion } from "motion/react";
import { Shape } from "@/components/visual/Shape";
import type { ThemeId } from "@/themes/tokens";
import { decorShapes } from "@/shapes/decor";

// Where the decoration shapes sit around the header, and their size. The top-right
// corner is left free for the share button.
const spots = [
  { className: "-left-2 top-6", size: 54, rotate: -14 },
  { className: "-right-3 top-24", size: 72, rotate: 10 },
  { className: "left-10 top-44", size: 38, rotate: 22 },
];

interface DecorProps {
  theme: ThemeId;
  seed: string;
}

/**
 * The shapes around the header. zazi: they float; papier: still stickers;
 * pop: bigger. The other themes rely on their animated background instead.
 */
export function Decor({ theme, seed }: DecorProps) {
  const reduce = useReducedMotion();
  if (theme !== "zazi" && theme !== "papier" && theme !== "pop") return null;
  const ids = decorShapes(seed, spots.length);
  const scale = theme === "pop" ? 1.35 : 1;

  return (
    <div aria-hidden="true" className="pointer-events-none absolute inset-x-0 top-0 h-56">
      {ids.map((id, i) => {
        const spot = spots[i]!;
        const floating = theme !== "papier" && !reduce;
        return (
          <m.div
            key={id}
            className={`absolute ${spot.className}`}
            initial={{ opacity: 0, scale: 0.6, rotate: spot.rotate }}
            animate={
              floating
                ? { opacity: 1, scale: 1, rotate: spot.rotate, y: [0, -8, 0] }
                : { opacity: 1, scale: 1, rotate: spot.rotate }
            }
            transition={{
              opacity: { duration: 0.6, delay: 0.2 + i * 0.1 },
              scale: { type: "spring", stiffness: 200, damping: 14, delay: 0.2 + i * 0.1 },
              y: { duration: 6 + i * 1.5, repeat: Infinity, ease: "easeInOut" },
            }}
          >
            <Shape
              id={id}
              size={Math.round(spot.size * scale)}
              className={theme === "papier" ? "lt-sticker" : "opacity-90"}
            />
          </m.div>
        );
      })}
    </div>
  );
}
