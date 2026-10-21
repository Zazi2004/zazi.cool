import Grainient from "@/components/reactbits/Grainient";
import { themes } from "@/themes/tokens";
import type { ThemeBackgroundProps } from "./AnimatedBackground";

/** pop: a lively grainy gradient, stirred by the pointer. */
export default function PopBackground({ accent }: ThemeBackgroundProps) {
  const colors = themes.pop.palettes(accent).light!.background;
  return (
    <Grainient
      colors={colors}
      bands={2}
      speed={0.8}
      flow={1}
      grainAmount={0.16}
      interactive
      clickSwirl={false}
      dpr={1.5}
    />
  );
}
