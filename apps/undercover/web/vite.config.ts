import { defineConfig } from "vitest/config";
import { loadEnv } from "vite";
import preact from "@preact/preset-vite";

export default defineConfig(({ mode }) => {
  // The Go server's PORT comes from the app's .env, one directory up.
  const env = loadEnv(mode, "..", "");
  return {
    plugins: [preact()],
    server: {
      // In development, Vite serves the front with hot reload and forwards API calls
      // to the Go server (`go run ./cmd/undercover`). 127.0.0.1 rather than localhost:
      // localhost may resolve to IPv6 first, where another local service can listen.
      proxy: { "/api": `http://127.0.0.1:${env.PORT || "8080"}` },
    },
    test: {
      environment: "node",
    },
  };
});
