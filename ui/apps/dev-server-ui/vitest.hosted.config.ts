import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['src/utils/devServer.test.ts', 'src/utils/hostedRoutes.test.ts'],
    environment: 'node',
  },
});
