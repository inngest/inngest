import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import { nitroV2Plugin } from '@tanstack/nitro-v2-vite-plugin';

import { defineConfig } from 'vite';
import tsConfigPaths from 'vite-tsconfig-paths';
import viteReact from '@vitejs/plugin-react';
import path from 'path';
import svgrPlugin from 'vite-plugin-svgr';

export default defineConfig({
  define: {
    'import.meta.env.VITE_VERCEL_GIT_COMMIT_SHA': JSON.stringify(
      process.env.VERCEL_GIT_COMMIT_SHA,
    ),
    'import.meta.env.VITE_VERCEL_ENV': JSON.stringify(process.env.VERCEL_ENV),
  },
  resolve: {
    alias: {
      '@inngest/components': path.resolve(
        __dirname,
        '../../packages/components/src',
      ),
    },
    // TODO: these can go away when all versions are aligned across monorepo
    dedupe: [
      'next-themes',
      '@tanstack/react-query',
      'react',
      'react-dom',
      '@tanstack/react-router',
      '@tanstack/react-table',
      'zod',
    ],
  },
  optimizeDeps: {
    exclude: ['@inngest/agent-kit'],
  },
  ssr: {
    noExternal: ['@headlessui/tailwindcss', 'react-use'],
    external: [
      'monaco-editor',
      '@monaco-editor/react',
      'node:stream',
      'node:stream/web',
      'node:async_hooks',
    ],
  },
  plugins: [
    tanstackStart(),
    nitroV2Plugin(),
    tsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
    viteReact(),
    svgrPlugin(),
  ],
});
