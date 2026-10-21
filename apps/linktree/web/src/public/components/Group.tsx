import { ChevronDown } from "lucide-react";
import { Accordion } from "radix-ui";
import type { CSSProperties } from "react";
import { PlatformIcon } from "@/components/visual/PlatformIcon";
import { Visual } from "@/components/visual/Visual";
import { cn } from "@/lib/cn";
import type { ShownGroup } from "../model";
import { DraftBadge } from "./DraftBadge";

interface GroupProps {
  group: ShownGroup;
  inert?: boolean;
}

/**
 * A group of links, shown as an accordion item (the Accordion root is in
 * LinksPage). Closed, it shows its visual, title, subtitle and the icons of its
 * platforms; open, its links in a grid.
 */
export function Group({ group, inert }: GroupProps) {
  return (
    <Accordion.Item
      value={group.id}
      className={cn("lt-card overflow-hidden", group.draft && "opacity-55")}
    >
      <Accordion.Header className="m-0">
        <Accordion.Trigger className="lt-focus group flex w-full cursor-pointer items-center gap-3 p-3 text-left">
          {group.visual && (
            <Visual
              visual={group.visual}
              size={52}
              imageClassName="rounded-[calc(var(--lt-radius)*0.6)]"
            />
          )}
          <span className="min-w-0 flex-1">
            <span className="flex items-center gap-2">
              <span className="leading-snug font-semibold break-words">{group.title}</span>
              {group.draft && <DraftBadge />}
            </span>
            {group.subtitle && (
              <span className="block text-sm leading-snug text-lt-muted">{group.subtitle}</span>
            )}
            <span
              aria-hidden="true"
              className="mt-1.5 flex gap-1.5 text-lt-muted transition-opacity group-data-[state=open]:opacity-0"
            >
              {group.links.slice(0, 6).map((l) => (
                <PlatformIcon key={l.id} url={l.url} className="size-3.5" />
              ))}
            </span>
          </span>
          <ChevronDown
            aria-hidden="true"
            className="size-5 shrink-0 text-lt-muted transition-transform duration-300 group-data-[state=open]:rotate-180"
          />
        </Accordion.Trigger>
      </Accordion.Header>
      <Accordion.Content className="lt-accordion-content">
        <ul className="m-0 grid list-none gap-2 p-3 pt-0 @sm:grid-cols-2">
          {group.links.map((l, i) => (
            // --lt-i staggers the reveal animation (see themes.css).
            <li key={l.id} className="lt-reveal" style={{ "--lt-i": i } as CSSProperties}>
              <a
                href={l.url}
                onClick={inert ? (e) => e.preventDefault() : undefined}
                className={cn(
                  "lt-focus flex min-h-11 items-center gap-2.5 rounded-[calc(var(--lt-radius)*0.65)] bg-lt-surface-strong px-3 py-2 text-sm font-medium no-underline transition-transform active:scale-[0.97]",
                  l.draft && "opacity-55",
                )}
              >
                {l.visual ? (
                  <Visual visual={l.visual} size={22} imageClassName="rounded-md" />
                ) : (
                  <PlatformIcon url={l.url} className="size-4.5" />
                )}
                <span className="min-w-0 flex-1 break-words">{l.title}</span>
                {l.draft && <DraftBadge />}
              </a>
            </li>
          ))}
        </ul>
      </Accordion.Content>
    </Accordion.Item>
  );
}
