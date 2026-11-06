import { defineConfig } from "vitest/config";
import { loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  // The Go server's PORT comes from the app's .env, one directory up.
  const env = loadEnv(mode, "..", "");
  const api = `http://127.0.0.1:${env.PORT || "8080"}`;
  return {
    plugins: [react()],
    build: {
      // The CSP allows no data: font, and keeping every asset a file keeps it simple.
      assetsInlineLimit: 0,
    },
    server: {
      // In development, Vite serves the front with hot reload and forwards API calls
      // to the Go server (`go run ./cmd/habits`). 127.0.0.1 rather than localhost:
      // localhost may resolve to IPv6 first, where another local service can listen.
      proxy: {
        "/api": {
          target: api,
          // The Go server refuses a POST whose Origin is not BASE_URL: present the
          // requests as coming from there.
          headers: { Origin: env.BASE_URL || api },
        },
      },
    },
    test: {
      environment: "node",
    },
  };
});
