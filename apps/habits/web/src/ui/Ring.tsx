import { motion } from "motion/react";
import { useId, type ReactNode } from "react";
import { prefersReducedMotion } from "./effects/motion";

interface Props {
  size: number;
  stroke: number;
  progress: number;
  /** The big ring of the day; otherwise tinted by the enclosing `.color-*` class. */
  gradient?: boolean;
  children?: ReactNode;
}

/** A progress ring that springs to its value, with optional content in its centre. */
export function Ring({ size, stroke, progress, gradient = false, children }: Props) {
  const id = useId();
  const centre = size / 2;
  const radius = (size - stroke) / 2;
  const length = 2 * Math.PI * radius;
  const transition = prefersReducedMotion()
    ? { duration: 0 }
    : ({ type: "spring", stiffness: 120, damping: 20 } as const);

  return (
    <span className="ring" style={{ width: size, height: size }}>
      <svg width={size} height={size} aria-hidden="true">
        {gradient && (
          <defs>
            <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0" stopColor="var(--ring-from)" />
              <stop offset="1" stopColor="var(--ring-to)" />
            </linearGradient>
          </defs>
        )}
        <circle className="ring-track" cx={centre} cy={centre} r={radius} strokeWidth={stroke} />
        <motion.circle
          className="ring-value"
          cx={centre}
          cy={centre}
          r={radius}
          strokeWidth={stroke}
          strokeDasharray={length}
          stroke={gradient ? `url(#${id})` : undefined}
          initial={false}
          // Hidden at 0: a round line cap would otherwise draw a dot on an empty ring.
          animate={{ strokeDashoffset: length * (1 - progress), opacity: progress > 0 ? 1 : 0 }}
          transition={transition}
        />
      </svg>
      {children && <span className="ring-centre">{children}</span>}
    </span>
  );
}
