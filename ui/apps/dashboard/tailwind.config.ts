import type { Config } from 'tailwindcss';
import defaultTheme from 'tailwindcss/defaultTheme';

export default {
  content: ['./src/**/*.{ts,tsx}'],
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
      gridTemplateColumns: {
        dashboard: '1fr 1fr 1fr 400px',
      },
      boxShadow: {
        'outline-primary-light':
          'inset 0 1px 0 0 rgba(255, 255, 255, 0.1), inset 0 0 0 1px rgba(255, 255, 255, 0.1), 0 1px 3px rgba(0, 0, 0, 0.2)',
        'outline-primary-dark':
          '0 0 0 0.5px rgba(0, 0, 0, 0.4), inset 0 1px 0 0 rgba(255, 255, 255, 0.1), inset 0 0 0 1px rgba(255, 255, 255, 0.1), 0 1px 3px rgba(0, 0, 0, 0.2)',
        'outline-secondary-dark':
          '0 0 0 0.5px rgba(0, 0, 0, 0.3), inset 0 1px 0 0 rgba(255, 255, 255, 0.1), inset 0 0 0 1px rgba(255, 255, 255, 0.01), 0 1px 3px rgba(0, 0, 0, 0.15)',
        'outline-secondary-light':
          '0 0 0 0.5px rgba(0, 0, 0, 0.15), inset 0 1px 0 0 rgba(255, 255, 255, 0.8), inset 0 0 0 1px rgba(255, 255, 255, 0.1), 0 1px 3px rgba(0, 0, 0, 0.15)',
        floating: '0 0 0 0.5px rgba(0, 0, 0, 0.1), 0 1px 2px rgba(255, 255, 255, 0.15)',
      },
      keyframes: {
        'slide-down-and-fade': {
          '0%': { opacity: '0', transform: 'translateY(-3px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-left-and-fade': {
          '0%': { opacity: '0', transform: 'translateX(2px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        'slide-up-and-fade': {
          '0%': { opacity: '0', transform: 'translateY(2px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-right-and-fade': {
          '0%': { opacity: '0', transform: 'translateX(2px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        shimmer: {
          '100%': {
            transform: 'translateX(100%)',
          },
        },
      },
      animation: {
        'slide-down-and-fade': 'slide-down-and-fade 400ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-left-and-fade': 'slide-left-and-fade 400ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-up-and-fade': 'slide-up-and-fade 400ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-right-and-fade': 'slide-right-and-fade 400ms cubic-bezier(0.16, 1, 0.3, 1)',
      },
    },
    textShadow: {
      default: '0 1px 2px rgba(0, 0, 0, 0.2)',
    },
  },
  plugins: [require('@tailwindcss/forms'), require('@headlessui/tailwindcss')],
} satisfies Config;
