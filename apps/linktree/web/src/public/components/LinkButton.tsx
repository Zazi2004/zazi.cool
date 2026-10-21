import { ArrowUpRight } from "lucide-react";
import { m } from "motion/react";
import { PlatformIcon } from "@/components/visual/PlatformIcon";
import { Visual } from "@/components/visual/Visual";
import { cn } from "@/lib/cn";
import type { ShownLink } from "../model";
import { DraftBadge } from "./DraftBadge";

interface LinkButtonProps {
  link: ShownLink;
  /** In the admin preview, links do not navigate. */
  inert?: boolean;
}

/** A top-level link: a full-width card with its visual, title and an arrow. */
export function LinkButton({ link, inert }: LinkButtonProps) {
  return (
    <m.a
      href={link.url}
      onClick={inert ? (e) => e.preventDefault() : undefined}
      whileHover={{ y: -2 }}
      whileTap={{ scale: 0.97 }}
      transition={{ type: "spring", stiffness: 500, damping: 30 }}
      className={cn(
        "lt-card lt-link lt-focus group flex min-h-15 items-center gap-3 px-3 py-2.5 no-underline",
        link.draft && "opacity-55",
      )}
    >
      <span className="flex size-10 shrink-0 items-center justify-center">
        {link.visual ? (
          <Visual
            visual={link.visual}
            size={40}
            imageClassName="rounded-[calc(var(--lt-radius)*0.55)]"
          />
        ) : (
          <PlatformIcon url={link.url} className="size-6" />
        )}
      </span>
      <span className="flex-1 leading-snug font-medium break-words">{link.title}</span>
      {link.draft && <DraftBadge />}
      <ArrowUpRight
        aria-hidden="true"
        className="size-4 shrink-0 text-lt-muted transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
      />
    </m.a>
  );
}
