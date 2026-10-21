// Entry point of the admin: a single-page app under /admin, in its own bundle so
// that visitors of the public pages never download it.
import "@vitejs/plugin-react/preamble";
import "@fontsource-variable/geist";
import "virtual:theme-tokens.css";
import "./admin.css";
import { setNonce } from "get-nonce";
import { LazyMotion, MotionConfig, domAnimation } from "motion/react";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";

// The dialogs inject a <style> tag to lock the page scroll: the CSP of the admin
// allows it with the nonce that the server put in the page (see internal/api).
const nonce = document.querySelector<HTMLMetaElement>('meta[name="csp-nonce"]')?.content;
if (nonce) setNonce(nonce);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <LazyMotion features={domAnimation} strict>
      <MotionConfig reducedMotion="user">
        <App />
      </MotionConfig>
    </LazyMotion>
  </StrictMode>,
);
