import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import viteReact from '@vitejs/plugin-react';
import path from 'path';

import tsConfigPaths from 'vite-tsconfig-paths';

import { Connect, defineConfig, type PreviewServer } from 'vite';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import http from 'http';

//
// Plugin to handle SPA routing and API proxying during `vite preview`.
// Proxies /v0/* requests to the Go dev server and serves the SPA shell for all other routes.
const previewPlugin = () => ({
  name: 'tanstack-start-preview-middleware',
  configurePreviewServer(server: PreviewServer) {
    const devServerUrl = 'http://localhost:8288';

    const middleware: Connect.NextHandleFunction = (req, res, next) => {
      //
      // Proxy API requests to the Go dev server
      if (req.url?.startsWith('/v0/') || req.url?.startsWith('/dev')) {
        const targetUrl = `${devServerUrl}${req.url}`;
        const proxyReq = http.request(
          targetUrl,
          {
            method: req.method,
            headers: { ...req.headers, host: new URL(devServerUrl).host },
          },
          (proxyRes) => {
            res.writeHead(proxyRes.statusCode ?? 500, proxyRes.headers);
            proxyRes.pipe(res);
          },
        );
        proxyReq.on('error', () => {
          res.writeHead(502);
          res.end(
            'API proxy error - is the Go dev server running on port 8288?',
          );
        });
        req.pipe(proxyReq);
        return;
      }

      //
      // Serve SPA shell for non-asset routes
      if (!req.url?.includes('.')) {
        try {
          const shellPath = resolve(__dirname, 'dist/client/_shell.html');
          const shellHtml = readFileSync(shellPath, 'utf-8');
          res.setHeader('Content-Type', 'text/html');
          res.end(shellHtml);
          return;
        } catch {
          // Fall through to default handler
        }
      }

      next();
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
    //
    // TODO: these can go away when all versions are aligned across monorepo
    dedupe: ['next-themes', '@tanstack/react-query', 'react', 'react-dom'],
  },
  ssr: {
    noExternal: ['@reduxjs/toolkit', '@rtk-query/graphql-request-base-query'],
  },
  plugins: [
    tsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
    tanstackStart({
      spa: { enabled: true },
    }),
    viteReact(),
    previewPlugin(),
  ],
});
