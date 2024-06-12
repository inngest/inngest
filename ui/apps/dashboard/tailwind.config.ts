import type { Config } from 'tailwindcss';

import sharedConfig from '@inngest/components/tailwind.config';

export default {
  ...sharedConfig,
  content: ['./src/**/*.{ts,tsx}', '../../packages/components/src/**/*.{ts,tsx}'],
  plugins: [require('@tailwindcss/forms'), require('@headlessui/tailwindcss')],
} satisfies Config;
