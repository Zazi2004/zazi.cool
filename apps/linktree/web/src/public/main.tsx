// Entry point of the public pages. The Go server renders the HTML (meta tags,
// data-theme, page data); this script draws the page from that data.
import "@vitejs/plugin-react/preamble";
import "virtual:theme-tokens.css";
import "./public.css";
import { LazyMotion, domAnimation, MotionConfig } from "motion/react";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { readPageData } from "@/shared/pageData";
import { loadThemeFonts } from "@/themes/fonts";
import { IndexPage } from "./IndexPage";
import { LinksPage } from "./LinksPage";
import { NotFoundPage } from "./NotFoundPage";

const data = readPageData();
void loadThemeFonts(data.view === "page" ? data.page.theme : "zazi");

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    {/* LazyMotion with domAnimation keeps Motion's bundle small; "user" respects
        prefers-reduced-motion everywhere. */}
    <LazyMotion features={domAnimation} strict>
      <MotionConfig reducedMotion="user">
        {data.view === "page" ? (
          <LinksPage page={data.page} />
        ) : data.view === "index" ? (
          <IndexPage pages={data.pages} />
        ) : (
          <NotFoundPage />
        )}
      </MotionConfig>
    </LazyMotion>
  </StrictMode>,
);
