import { ArrowUpRight } from "lucide-react";
import { m } from "motion/react";
import type { IndexEntry } from "@/api/types";
import { Shape } from "@/components/visual/Shape";
import { Visual } from "@/components/visual/Visual";
import { Backdrop } from "./components/Backdrop";
import { Decor } from "./components/Decor";
import { Footer } from "./components/Footer";
import { Title } from "./components/Title";

/** The index (/): every published, listed page. */
export function IndexPage({ pages }: { pages: IndexEntry[] }) {
  return (
    <div className="lt-page @container relative min-h-dvh">
      <Backdrop theme="zazi" accent="violet" />
      <div className="relative mx-auto w-full max-w-[40rem] px-4">
        <Decor theme="zazi" seed="index" />
        <header className="flex flex-col items-center gap-3 pt-16 pb-10 text-center">
          <p className="m-0 text-sm font-medium tracking-[0.2em] text-lt-muted uppercase">
            zazi.cool
          </p>
          <Title text="Les pages de liens" theme="zazi" />
        </header>

        {pages.length === 0 ? (
          <div className="flex flex-col items-center gap-4 py-12 text-center text-lt-muted">
            <Shape id="moon-3" size={96} />
            <p className="m-0">Aucune page publiée pour l'instant.</p>
          </div>
        ) : (
          <ul className="m-0 grid list-none gap-3 p-0 @lg:grid-cols-2">
            {pages.map((p, i) => (
              <m.li
                key={p.slug}
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.35, delay: 0.25 + i * 0.05 }}
              >
                <m.a
                  href={`/${p.slug}`}
                  whileHover={{ y: -3 }}
                  whileTap={{ scale: 0.98 }}
                  className="lt-card lt-link lt-focus group flex h-full items-center gap-4 p-4 no-underline"
                >
                  {p.avatar ? (
                    <Visual visual={p.avatar} size={56} imageClassName="rounded-full" />
                  ) : (
                    <span className="size-14 shrink-0" />
                  )}
                  <span className="min-w-0 flex-1">
                    <span className="lt-title block text-lg font-semibold">{p.title}</span>
                    {p.description && (
                      <span className="line-clamp-2 text-sm text-lt-muted">{p.description}</span>
                    )}
                  </span>
                  <ArrowUpRight aria-hidden="true" className="size-4 shrink-0 text-lt-muted" />
                </m.a>
              </m.li>
            ))}
          </ul>
        )}
        <Footer />
      </div>
    </div>
  );
}
