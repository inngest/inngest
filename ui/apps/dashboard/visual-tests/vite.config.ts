import path from 'node:path';
import { fileURLToPath } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

const visualTestsDir = path.dirname(fileURLToPath(import.meta.url));
const dashboardDir = path.resolve(visualTestsDir, '..');

export default defineConfig({
  root: visualTestsDir,
  build: {
    outDir: path.resolve(dashboardDir, 'test-results/visual-fixture'),
  },
  resolve: {
    alias: {
      '@': path.resolve(dashboardDir, 'src'),
      '@inngest/components': path.resolve(
        dashboardDir,
        '../../packages/components/src',
      ),
    },
    dedupe: ['react', 'react-dom'],
  },
  css: {
    postcss: path.resolve(dashboardDir, 'postcss.config.mjs'),
  },
  plugins: [react()],
});
