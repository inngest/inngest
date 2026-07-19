import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import { nitroV2Plugin } from '@tanstack/nitro-v2-vite-plugin';

import { defineConfig } from 'vite';
import tsConfigPaths from 'vite-tsconfig-paths';
import viteReact from '@vitejs/plugin-react';
import path from 'path';
import svgrPlugin from 'vite-plugin-svgr';

/**
 * Demo build variant. Mirrors vite.config.ts but:
 *  - flags VITE_DEMO_MODE so the app bypasses auth wiring and the /gql route
 *    serves the fake App API (src/demo/mock),
 *  - aliases the Clerk SDK (client + server) and the LaunchDarkly client SDK to
 *    local fakes so no real auth/flag service is contacted and no sign-in is
 *    required.
 * The production config is untouched, keeping the real dashboard build safe.
 */
export default defineConfig({
  define: {
    'import.meta.env.VITE_VERCEL_GIT_COMMIT_SHA': JSON.stringify(
      process.env.VERCEL_GIT_COMMIT_SHA,
    ),
    'import.meta.env.VITE_VERCEL_ENV': JSON.stringify(process.env.VERCEL_ENV),
    'import.meta.env.VITE_DEMO_MODE': JSON.stringify('true'),
  },
  resolve: {
    // Array form for exact (`^...$`) matching so the client and `/server`
    // specifiers map to distinct stubs.
    alias: [
      {
        find: /^@clerk\/tanstack-react-start$/,
        replacement: path.resolve(__dirname, 'src/demo/clerk/client.tsx'),
      },
      {
        find: /^@clerk\/tanstack-react-start\/server$/,
        replacement: path.resolve(__dirname, 'src/demo/clerk/server.ts'),
      },
      {
        find: /^launchdarkly-react-client-sdk$/,
        replacement: path.resolve(__dirname, 'src/demo/ld/client.tsx'),
      },
      {
        find: '@inngest/components',
        replacement: path.resolve(__dirname, '../../packages/components/src'),
      },
    ],
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
    // Do not prebundle Clerk here — it is aliased to local source.
    exclude: ['@inngest/agent-kit', '@clerk/tanstack-react-start'],
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
  server: {
    host: true,
  },
});
