import path from 'path';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@inngest/components': path.resolve(__dirname, 'src'),
    },
  },
  test: {
    environment: 'jsdom',
  },
});
