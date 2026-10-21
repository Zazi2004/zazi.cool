import Aurora from "@/components/reactbits/Aurora";
import { accents, themes } from "@/themes/tokens";
import type { ThemeBackgroundProps } from "./AnimatedBackground";

/** nuit: an aurora in the accent color and cyan, seen through the theme opacity. */
export default function NuitBackground({ accent }: ThemeBackgroundProps) {
  const bright = accents[accent].bright;
  return (
    // The contrast tests assume the aurora is drawn at exactly this opacity.
    <div className="absolute inset-0" style={{ opacity: themes.nuit.animatedOpacity }}>
      <Aurora
        colorStops={[bright, "#22d3ee", bright]}
        amplitude={1.1}
        blend={0.6}
        speed={0.6}
        height={0.55}
        layers={1}
        mouseInteraction={false}
        dpr={1.5}
      />
    </div>
  );
}
