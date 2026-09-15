import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';
import tsConfigPaths from 'vite-tsconfig-paths';

export default defineConfig({
  resolve: {
    alias: {
      '@inngest/components': fileURLToPath(
        new URL('../../packages/components/src', import.meta.url),
      ),
    },
  },
  plugins: [
    tsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
  ],
  test: {
    environment: 'node',
  },
});
