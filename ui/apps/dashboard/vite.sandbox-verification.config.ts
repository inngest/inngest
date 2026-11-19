import path from 'node:path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Isolated UI review: no auth, verification APIs, analytics, or production routes.
export default defineConfig({
  resolve: {
    alias: {
      '@inngest/components': path.resolve(
        __dirname,
        '../../packages/components/src',
      ),
      '@': path.resolve(__dirname, 'src'),
    },
    dedupe: ['react', 'react-dom', '@tanstack/react-router'],
  },
  server: {
    host: '127.0.0.1',
    port: 5175,
    open: '/sandbox-verification-demo.html',
  },
  plugins: [
    {
      name: 'sandbox-verification-demo-analytics',
      enforce: 'pre',
      resolveId(source) {
        if (source === '@/utils/analyticsEvents')
          return '\0sandbox-demo-analytics';
      },
      load(id) {
        if (id === '\0sandbox-demo-analytics')
          return 'export function trackEmptyStateViewed() {} export function trackEmptyStateUpgradeClicked() {}';
      },
    },
    react(),
    {
      name: 'sandbox-verification-demo-development-only',
      config(_, { command }) {
        if (command !== 'serve')
          throw new Error('The sandbox verification demo is development-only.');
      },
    },
  ],
});
