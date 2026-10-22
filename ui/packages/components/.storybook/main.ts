import { createRequire } from 'node:module';
import { dirname } from 'node:path';
import type { StorybookConfig } from '@storybook/nextjs';

const require = createRequire(import.meta.url);

const config: StorybookConfig = {
  stories: ['../src/**/*.mdx', '../src/**/*.stories.@(js|jsx|mjs|ts|tsx)'],
  addons: [
    '@storybook/addon-a11y',
    '@storybook/addon-links',
    '@storybook/addon-themes',
    '@storybook/addon-docs',
  ],
  framework: {
    name: '@storybook/nextjs',
    options: {},
  },
  webpackFinal: async (config) => {
    const alias = { ...config.resolve?.alias };
    // Use the components' React 18, not the Next framework's bundled React 19.
    for (const key of Object.keys(alias)) {
      if (/^react(?:-dom)?(?:\$|\/|$)/.test(key)) {
        delete alias[key];
      }
    }
    config.resolve = {
      ...config.resolve,
      alias: {
        ...alias,
        react: dirname(require.resolve('react/package.json')),
        'react-dom': dirname(require.resolve('react-dom/package.json')),
      },
    };
    return config;
  },
};
export default config;
