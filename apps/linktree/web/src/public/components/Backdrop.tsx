import { AnimatedBackground } from "@/backgrounds/AnimatedBackground";
import type { AccentId, ThemeId } from "@/themes/tokens";

interface BackdropProps {
  theme: ThemeId;
  accent: AccentId;
}

/**
 * The page background: the theme gradient, the animated background of the theme
 * (if any), then the grain. Fixed to the viewport; in the admin preview, the
 * frame's transform makes "fixed" relative to the frame instead.
 */
export function Backdrop({ theme, accent }: BackdropProps) {
  return (
    <div aria-hidden="true" className="pointer-events-none fixed inset-0 -z-10">
      <div className="lt-backdrop absolute inset-0" />
      <AnimatedBackground theme={theme} accent={accent} />
      <div className="lt-grain absolute inset-0" />
    </div>
  );
}
