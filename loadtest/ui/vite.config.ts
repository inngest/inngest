import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";

export default defineConfig({
  plugins: [TanStackRouterVite(), react()],
  build: {
    outDir: "../internal/uiembed/dist",
    emptyOutDir: true,
  },
  server: {
    port: 9011,
    proxy: {
      "/api": "http://127.0.0.1:9010",
    },
  },
});
