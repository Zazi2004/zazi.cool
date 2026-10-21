import { fileURLToPath, URL } from "node:url";
import type { Plugin } from "vite";
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { themeCss } from "./src/themes/css";

// Serves the CSS variables of the themes, generated from src/themes/tokens.ts, as
// the module "virtual:theme-tokens.css": one definition for the CSS, the contrast
// tests and the animated backgrounds.
function themeTokens(): Plugin {
  const id = "virtual:theme-tokens.css";
  // The "\0" prefix marks a virtual module for the other plugins; the .css suffix
  // makes Vite treat it as a stylesheet.
  const resolved = "\0theme-tokens.css";
  return {
    name: "theme-tokens",
    resolveId: (source) => (source === id ? resolved : undefined),
    load: (source) => (source === resolved ? themeCss() : undefined),
  };
}

// The Go server renders the HTML pages and loads these entry points: Vite only
// builds the bundles (see internal/render). In development, the pages served by Go
// load them from this dev server, with hot reload.
export default defineConfig({
  plugins: [react(), tailwindcss(), themeTokens()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  build: {
    // Go reads .vite/manifest.json to find the hashed file names.
    manifest: true,
    rollupOptions: {
      input: {
        public: "src/public/main.tsx",
        admin: "src/admin/main.tsx",
      },
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    // URLs of assets imported by CSS/JS must point to this server, not to Go's.
    origin: "http://localhost:5173",
  },
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
  },
});
