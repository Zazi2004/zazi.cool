import Threads from "@/components/reactbits/Threads";
import { accents, themes } from "@/themes/tokens";
import type { ThemeBackgroundProps } from "./AnimatedBackground";

/** neon: luminous threads in the accent color, seen through the theme opacity. */
export default function NeonBackground({ accent }: ThemeBackgroundProps) {
  return (
    // The contrast tests assume the threads are drawn at exactly this opacity.
    <div className="absolute inset-0" style={{ opacity: themes.neon.animatedOpacity }}>
      <Threads
        color={accents[accent].bright}
        amplitude={1}
        distance={0.2}
        speed={0.5}
        enableMouseInteraction={false}
      />
    </div>
  );
}
