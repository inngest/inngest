import tailwindcss from '@tailwindcss/vite';
import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import viteReact from '@vitejs/plugin-react';
import mdx from 'fumadocs-mdx/vite';
import { defineConfig } from 'vite';
import tsConfigPaths from 'vite-tsconfig-paths';

export default defineConfig(async () => ({
  resolve: {
    alias: {
      // fumadocs-openapi/ui uses next/dynamic for lazy-loading APIPlayground.
      // Shim it with React.lazy so it works outside of Next.js.
      'next/dynamic': new URL('./src/shims/next-dynamic.ts', import.meta.url).pathname,
      // fumadocs-mdx/runtime/server imports node:path at the top level. Rollup
      // tree-shakes it away in production, but Vite dev pre-bundles it with
      // esbuild which evaluates it in browser context. Shim it so the module
      // initialises cleanly in SPA dev mode.
      'node:path': new URL('./src/shims/node-path.ts', import.meta.url).pathname,
    },
  },
  plugins: [
    tsConfigPaths({ projects: ['./tsconfig.json'] }),
    mdx(await import('./source.config')),
    tailwindcss(),
    tanstackStart({
      spa: { enabled: true },
    }),
    viteReact(),
  ],
}));
