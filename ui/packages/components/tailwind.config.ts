import type { Config } from 'tailwindcss';
import defaultTheme from 'tailwindcss/defaultTheme';

export default {
  content: ['./src/**/*.{ts,tsx,mdx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['var(--font-inter-tight)', ...defaultTheme.fontFamily.sans],
        mono: ['var(--font-roboto-mono)', ...defaultTheme.fontFamily.mono],
      },
      colors: {
        slate: {
          910: '#0C1323',
          940: '#080D19',
        },
      },
      boxShadow: {
        'outline-primary-light':
          'inset 0 1px 0 0 rgba(255, 255, 255, 0.1), inset 0 0 0 1px rgba(255, 255, 255, 0.1), 0 1px 3px rgba(0, 0, 0, 0.2)',
      },
      outlineOffset: {
        3: '3px',
      },
      keyframes: {
        shimmer: {
          '100%': {
            transform: 'translateX(100%)',
          },
        },
        'slide-down-and-fade': {
          '0%': { opacity: '0', transform: 'translateY(-3px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
      animation: {
        'slide-down-and-fade': 'slide-down-and-fade 0.2s cubic-bezier(0, 1, 0.3, 1)',
      },
    },
  },
  plugins: [require('@headlessui/tailwindcss')],
} satisfies Config;
