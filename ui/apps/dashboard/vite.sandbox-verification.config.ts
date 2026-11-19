import path from 'node:path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Isolated UI review: no auth, verification APIs, analytics, or production routes.
export default defineConfig({
  resolve: {
    alias: {
      '@/utils/analyticsEvents': path.resolve(
        __dirname,
        'src/components/Sandboxes/verification-demo/analytics.ts',
      ),
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
