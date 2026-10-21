import { m, type Variants } from "motion/react";
import { Accordion } from "radix-ui";
import { Backdrop } from "./components/Backdrop";
import { Decor } from "./components/Decor";
import { Footer } from "./components/Footer";
import { Group } from "./components/Group";
import { LinkButton } from "./components/LinkButton";
import { PageHeader } from "./components/PageHeader";
import { ShareButton } from "./components/ShareButton";
import type { ShownItem, ShownPage } from "./model";

// Items arrive one after the other, once, right after the header.
const list: Variants = {
  hidden: {},
  shown: { transition: { staggerChildren: 0.045, delayChildren: 0.3 } },
};
const entry: Variants = {
  hidden: { opacity: 0, y: 12, filter: "blur(4px)" },
  shown: { opacity: 1, y: 0, filter: "blur(0px)", transition: { duration: 0.35, ease: "easeOut" } },
};

interface LinksPageProps {
  page: ShownPage;
  /** Admin preview: links do not navigate, sharing is off. */
  preview?: boolean;
}

/** A links page: header, links and groups (accordions), footer. */
export function LinksPage({ page, preview }: LinksPageProps) {
  const openByDefault = page.items
    .filter((it) => it.kind === "group" && it.openByDefault)
    .map((it) => it.id);
  // Several groups can be open at once: the accordion is of type "multiple". Its
  // key resets the open groups when the defaults change in the admin preview.
  return (
    <div className="lt-page @container relative min-h-dvh">
      <Backdrop theme={page.theme} accent={page.accent} />
      <div className="relative mx-auto w-full max-w-[34rem] px-4 pb-4">
        <Decor theme={page.theme} seed={page.id} />
        <div className="absolute top-4 right-4 z-10">
          <ShareButton title={page.title} disabled={preview} />
        </div>
        <PageHeader
          title={page.title}
          description={page.description}
          avatar={page.avatar}
          theme={page.theme}
        />
        <Accordion.Root
          type="multiple"
          defaultValue={openByDefault}
          key={openByDefault.join()}
          asChild
        >
          <m.ul
            variants={list}
            initial="hidden"
            animate="shown"
            className="m-0 mt-8 flex list-none flex-col gap-3 p-0"
          >
            {page.items.map((item) => (
              <m.li key={item.id} variants={entry}>
                <Item item={item} inert={preview} />
              </m.li>
            ))}
          </m.ul>
        </Accordion.Root>
        <Footer />
      </div>
    </div>
  );
}

function Item({ item, inert }: { item: ShownItem; inert?: boolean }) {
  return item.kind === "group" ? (
    <Group group={item} inert={inert} />
  ) : (
    <LinkButton link={item} inert={inert} />
  );
}
