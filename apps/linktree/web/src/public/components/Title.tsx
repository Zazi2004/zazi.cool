import { m, useReducedMotion } from "motion/react";
import type { ThemeId } from "@/themes/tokens";

interface TitleProps {
  text: string;
  theme: ThemeId;
}

/**
 * The page title with the entrance effect of its theme: words that come out of a
 * blur (the BlurText idea of React Bits), and a light sweeping over the letters
 * for neon (ShinyText). Without motion, the plain title.
 */
export function Title({ text, theme }: TitleProps) {
  const reduce = useReducedMotion();
  const classes =
    "lt-title m-0 text-[2rem] leading-[1.1] font-bold tracking-tight text-balance break-words";

  if (reduce) return <h1 className={classes}>{text}</h1>;
  if (theme === "neon") {
    return <h1 className={`${classes} lt-shiny`}>{text}</h1>;
  }
  const words = text.split(" ");
  return (
    <h1 className={classes} aria-label={text}>
      {words.map((word, i) => (
        <m.span
          key={i}
          aria-hidden="true"
          className="inline-block whitespace-pre"
          initial={{ opacity: 0, filter: "blur(10px)", y: 8 }}
          animate={{ opacity: 1, filter: "blur(0px)", y: 0 }}
          transition={{ duration: 0.5, delay: 0.15 + i * 0.07, ease: "easeOut" }}
        >
          {word + (i < words.length - 1 ? " " : "")}
        </m.span>
      ))}
    </h1>
  );
}
