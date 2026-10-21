import Grainient from "@/components/reactbits/Grainient";
import { themes } from "@/themes/tokens";
import type { ThemeBackgroundProps } from "./AnimatedBackground";

/** papier: a slow, pale grainy gradient in the colors of the palette. */
export default function PapierBackground({ accent }: ThemeBackgroundProps) {
  const colors = themes.papier.palettes(accent).light!.background;
  return (
    <Grainient
      colors={colors}
      speed={0.35}
      flow={0.5}
      grainAmount={0.12}
      softness={1}
      interactive={false}
      clickSwirl={false}
      dpr={1.5}
    />
  );
}
