import { m } from "motion/react";
import { Shape } from "@/components/visual/Shape";
import { digitShapes } from "@/shapes/decor";
import { Backdrop } from "./components/Backdrop";

/** 404: the number spelled with coolshapes digits. */
export function NotFoundPage() {
  return (
    <div className="lt-page relative flex min-h-dvh flex-col items-center justify-center gap-8 px-4 text-center">
      <Backdrop theme="zazi" accent="violet" />
      <div className="flex gap-1" role="img" aria-label="404">
        {digitShapes("404").map((id, i) => (
          <m.div
            key={i}
            initial={{ opacity: 0, y: 30, rotate: -10 }}
            animate={{ opacity: 1, y: 0, rotate: 0 }}
            transition={{ type: "spring", stiffness: 180, damping: 12, delay: i * 0.12 }}
          >
            <Shape id={id} size={96} />
          </m.div>
        ))}
      </div>
      <div className="flex flex-col items-center gap-2">
        <h1 className="lt-title m-0 text-2xl font-bold">Cette page n'existe pas (ou plus).</h1>
        <p className="m-0 text-lt-muted">Elle a peut-être changé d'adresse.</p>
      </div>
      <a
        href="/"
        className="lt-card lt-link lt-focus rounded-full px-5 py-2.5 font-medium no-underline"
      >
        Voir toutes les pages
      </a>
    </div>
  );
}
