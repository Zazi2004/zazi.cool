import { defineConfig } from "vitest/config";

// The Go server renders the page and embeds web/dist (see embed.go). Vite builds the
// bootstrap script and one module per animated background (a dynamic import), with
// content-hashed names: Go serves them as they are, and reads .vite/manifest.json to
// find the bootstrap script.
export default defineConfig({
  build: {
    target: "es2022",
    manifest: true,
    // No preload helper: the bootstrap imports a single module, once.
    modulePreload: false,
    rollupOptions: { input: { boot: "src/boot.ts" } },
  },
  test: {
    environment: "node",
    include: ["src/**/*.test.ts", "scripts/**/*.test.ts"],
  },
});
