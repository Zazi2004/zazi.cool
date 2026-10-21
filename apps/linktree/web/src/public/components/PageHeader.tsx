import { m } from "motion/react";
import type { Visual as VisualData } from "@/api/types";
import { Visual } from "@/components/visual/Visual";
import type { ThemeId } from "@/themes/tokens";
import { Title } from "./Title";

interface PageHeaderProps {
  title: string;
  description: string;
  avatar: VisualData | null;
  theme: ThemeId;
}

/** Avatar (image or shape), title and description. */
export function PageHeader({ title, description, avatar, theme }: PageHeaderProps) {
  const shapeSize = theme === "pop" ? 150 : 116;
  return (
    <header className="relative flex flex-col items-center gap-4 pt-10 text-center">
      {avatar && (
        <m.div
          initial={{ opacity: 0, scale: 0.7 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ type: "spring", stiffness: 260, damping: 18 }}
        >
          {avatar.kind === "image" ? (
            <Visual
              visual={avatar}
              size={104}
              eager
              imageClassName="rounded-full ring-4 ring-lt-surface shadow-lg"
            />
          ) : (
            <Visual visual={avatar} size={shapeSize} eager className="drop-shadow-xl" />
          )}
        </m.div>
      )}
      <Title text={title} theme={theme} />
      {description && (
        <m.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.35 }}
          className="m-0 max-w-[30ch] text-[1.02rem] leading-relaxed whitespace-pre-line text-lt-muted"
        >
          {description}
        </m.p>
      )}
    </header>
  );
}
