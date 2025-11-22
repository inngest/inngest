import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import { nitroV2Plugin } from "@tanstack/nitro-v2-vite-plugin";

import { defineConfig } from "vite";
import tsConfigPaths from "vite-tsconfig-paths";
import viteReact from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  resolve: {
    alias: {
      "@inngest/components": path.resolve(
        __dirname,
        "../../packages/components/src",
      ),
    },
    // TANSTACK TODO: these can go away when dashboard is converted and versions are in line
    dedupe: [
      "next-themes",
      "@tanstack/react-query",
      "react",
      "react-dom",
      "@tanstack/react-router",
      "@tanstack/react-table",
    ],
  },
  ssr: {
    noExternal: ["@headlessui/tailwindcss"],
    external: [
      "monaco-editor",
      "@monaco-editor/react",
      "node:stream",
      "node:stream/web",
      "node:async_hooks",
    ],
  },
  plugins: [
    tanstackStart(),
    nitroV2Plugin(),
    tsConfigPaths({
      projects: ["./tsconfig.json"],
    }),
    viteReact(),
  ],
});
