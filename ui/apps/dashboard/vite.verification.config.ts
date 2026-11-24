import path from 'node:path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

/** UI review with local navigation and service fixtures; no live integrations. */
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
  server: { host: '127.0.0.1', port: 5174, open: '/verification-demo.html' },
  plugins: [
    {
      name: 'verification-demo-shell-services',
      enforce: 'pre',
      resolveId(source, importer) {
        if (
          source.endsWith('/utils/useGraphQLQuery') ||
          source.endsWith('/Onboarding/useOnboardingStep') ||
          source === '@clerk/tanstack-react-start' ||
          (importer?.includes('/NavigationV2/QuickSearch/') &&
            (source === '@/queries' ||
              source.endsWith('/src/queries') ||
              source.endsWith('/src/queries/index.ts') ||
              source === './data' ||
              source.endsWith('/QuickSearch/data') ||
              source.endsWith('/QuickSearch/data.ts')))
        ) {
          return path.resolve(
            __dirname,
            'src/components/ComputeVerification/demo/shellServices.ts',
          );
        }
      },
    },
    react(),
    {
      name: 'verification-demo-development-only',
      config(_, { command }) {
        if (command !== 'serve')
          throw new Error('The verification demo is development-only.');
      },
    },
  ],
});
