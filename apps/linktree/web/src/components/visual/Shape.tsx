import { cn } from "@/lib/cn";

interface ShapeProps {
  id: string;
  size: number;
  className?: string;
  /** Decorative by default: the shape says nothing a screen reader needs. */
  alt?: string;
}

/**
 * A coolshapes shape. Shown as an <img> rather than inline SVG: no clash between
 * the SVG ids of several shapes, no inline style hitting the CSP, shared cache.
 */
export function Shape({ id, size, className, alt = "" }: ShapeProps) {
  return (
    <img
      src={`/shapes/${id}.svg`}
      alt={alt}
      width={size}
      height={size}
      draggable={false}
      decoding="async"
      className={cn("pointer-events-none select-none", className)}
    />
  );
}
