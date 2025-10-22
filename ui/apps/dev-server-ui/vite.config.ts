import { tanstackStart } from '@tanstack/react-start/plugin/vite';
import viteReact from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig } from 'vite';
import tsConfigPaths from 'vite-tsconfig-paths';

export default defineConfig({
  resolve: {
    alias: {
      '@inngest/components': path.resolve(
        __dirname,
        '../../packages/components/src',
      ),
      '@tanstack/react-query': path.resolve(
        __dirname,
        './node_modules/@tanstack/react-query',
      ),
    },
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
