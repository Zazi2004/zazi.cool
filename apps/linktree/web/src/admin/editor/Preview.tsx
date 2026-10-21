import { useEffect } from "react";
import type { Document } from "@/api/types";
import { LinksPage } from "@/public/LinksPage";
import { loadThemeFonts } from "@/themes/fonts";
import { Switch } from "../ui/Switch";
import { toShownPage } from "./document";

interface PreviewProps {
  doc: Document;
  showDrafts: boolean;
  onShowDrafts: (show: boolean) => void;
}

/**
 * The page as visitors will see it, drawn by the public components from the editor
 * state, in a phone-sized frame. The frame's transform makes the "fixed" background
 * of the page relative to the frame, and data-theme scopes the theme to it.
 */
export function Preview({ doc, showDrafts, onShowDrafts }: PreviewProps) {
  useEffect(() => {
    void loadThemeFonts(doc.theme);
  }, [doc.theme]);

  return (
    <div className="flex flex-col items-center gap-3">
      <div
        data-theme={doc.theme}
        data-accent={doc.accent}
        className="relative h-[min(780px,calc(100dvh-10rem))] w-full max-w-[390px] [transform:translateZ(0)] overflow-hidden rounded-[2.5rem] border-[6px] border-zinc-900 bg-(--lt-bg-base) shadow-2xl dark:border-zinc-700"
      >
        <div className="h-full overflow-y-auto">
          <LinksPage page={toShownPage(doc, showDrafts)} preview />
        </div>
      </div>
      <div className="w-full max-w-[390px]">
        <Switch label="Afficher les brouillons" checked={showDrafts} onChange={onShowDrafts} />
      </div>
    </div>
  );
}
