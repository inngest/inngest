import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import viteReact from '@vitejs/plugin-react';
import path from 'path';

import tsConfigPaths from 'vite-tsconfig-paths';

import { Connect, defineConfig, PreviewServer } from 'vite';
import { resolve } from 'path';
import { readFileSync } from 'fs';

const startPreviewPlugin = () => ({
  name: 'tanstack-start-preview-middleware',
  configurePreviewServer(server: PreviewServer) {
    const shellPath = resolve(__dirname, 'dist/client/_shell.html');
    const shellHtml = readFileSync(shellPath, 'utf-8');

    const middleware: Connect.NextHandleFunction = (req, res, next) => {
      if (req.url?.includes('.') || req.url?.startsWith('/api/')) {
        return next();
      }
      res.setHeader('Content-Type', 'text/html');
      res.end(shellHtml);
    };

    server.middlewares.use(middleware);
  },
});

export default defineConfig({
  resolve: {
    alias: {
      '@inngest/components': path.resolve(
        __dirname,
        '../../packages/components/src',
      ),
    },
  },
  ssr: {
    noExternal: ['@reduxjs/toolkit', '@rtk-query/graphql-request-base-query'],
  },
  plugins: [
    startPreviewPlugin(),
    tsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
    tanstackStart({
      spa: {
        enabled: true,
        prerender: {
          crawlLinks: true,
        },
      },
    }),
    viteReact(),
  ],
});
