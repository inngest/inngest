import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import { defineConfig } from "vite";
import tsConfigPaths from "vite-tsconfig-paths";
import { nitro } from "nitro/vite";
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
  },
  ssr: {
    noExternal: ["@headlessui/tailwindcss"],
    external: ["next"],
  },
  plugins: [
    tsConfigPaths({
      projects: ["./tsconfig.json"],
    }),
    tanstackStart(),
    nitro(),
    viteReact(),
  ],
});
