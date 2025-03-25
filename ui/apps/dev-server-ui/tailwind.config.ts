import type { Config } from 'tailwindcss';

import sharedConfig from '../../packages/components/tailwind.config';

export default {
  ...sharedConfig,
  content: ['./src/**/*.{ts,tsx}', '../../packages/components/src/**/*.{ts,tsx}'],
  darkMode: 'class',
  plugins: [require('@headlessui/tailwindcss')],
} satisfies Config;
