import type { Config } from 'tailwindcss';
import defaultTheme from 'tailwindcss/defaultTheme';

import sharedConfig from '../../packages/components/tailwind.config';

export default {
  ...sharedConfig,
  theme: {
    extend: {
      ...sharedConfig.theme.extend,
      fontFamily: {
        sans: ['var(--font-inter-tight)', ...defaultTheme.fontFamily.sans],
        mono: ['var(--font-roboto-mono)', ...defaultTheme.fontFamily.mono],
      },
    },
  },
  content: ['./src/**/*.{ts,tsx}', '../../packages/components/src/**/*.{ts,tsx}'],
  darkMode: 'class',
  plugins: [require('@headlessui/tailwindcss')],
} satisfies Config;
