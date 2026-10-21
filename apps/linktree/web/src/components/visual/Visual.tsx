import type { Visual as VisualData } from "@/api/types";
import { cn } from "@/lib/cn";
import { Shape } from "./Shape";

interface VisualProps {
  visual: VisualData;
  size: number;
  /** Rounding of images (shapes keep their own outline). */
  imageClassName?: string;
  className?: string;
  eager?: boolean;
}

/** An uploaded image (cropped to a square) or a coolshapes shape. */
export function Visual({ visual, size, imageClassName, className, eager }: VisualProps) {
  if (visual.kind === "shape") {
    return <Shape id={visual.shape} size={size} className={cn("shrink-0", className)} />;
  }
  return (
    <img
      src={visual.url}
      alt=""
      width={size}
      height={size}
      loading={eager ? "eager" : "lazy"}
      decoding="async"
      className={cn("shrink-0 object-cover", imageClassName, className)}
      style={{ width: size, height: size }}
    />
  );
}
