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
