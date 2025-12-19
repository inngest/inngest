import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import { nitroV2Plugin } from '@tanstack/nitro-v2-vite-plugin';

import { defineConfig } from 'vite';
import tsConfigPaths from 'vite-tsconfig-paths';
import viteReact from '@vitejs/plugin-react';
import path from 'path';
import svgrPlugin from 'vite-plugin-svgr';
import fs from 'fs';

export default defineConfig({
  resolve: {
    alias: {
      '@inngest/components': path.resolve(
        __dirname,
        '../../packages/components/src',
      ),
    },
    // TANSTACK TODO: these can go away when dashboard is converted and versions are in line
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
    nitroV2Plugin({
      hooks: {
        compiled(nitro) {
          //
          // Hack alert: we're inlining the server side senty init here
          // so we don't have to eject from the nitro preset on vercel to do it the
          // sentry way: https://docs.sentry.io/platforms/javascript/guides/tanstackstart-react/
          // when we move to proper nitro we can use a plugin
          const entryPath = path.join(
            nitro.options.output.serverDir,
            'index.mjs',
          );

          if (fs.existsSync(entryPath)) {
            const content = fs.readFileSync(entryPath, 'utf-8');
            const sentryInit = `import * as Sentry from "@sentry/tanstackstart-react";
              Sentry.init({
                dsn: process.env.VITE_SENTRY_DSN,
                tracesSampleRate: 0.2,
              });
            `;

            if (!content.includes('Sentry.init')) {
              fs.writeFileSync(entryPath, sentryInit + content);
            }
          }
        },
      },
    }),
    tsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
    viteReact(),
    svgrPlugin(),
  ],
});
