// @ts-check
const defaultTheme = require('tailwindcss/defaultTheme');

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        mono: ['var(--font-roboto-mono)', ...defaultTheme.fontFamily.mono],
      },
      colors: {
        slate: {
          910: '#0C1323',
          940: '#080D19',
        },
      },
      outlineOffset: {
        3: '3px',
      },
      keyframes: {
        'pulse-spin': {
          '0%': {
            transform: 'rotate(0deg)',
          },
          '50%': {
            transform: 'rotate(360deg)',
          },
          '100%': {
            transform: 'rotate(360deg)',
          },
        },
        shimmer: {
          '100%': {
            transform: 'translateX(100%)',
          },
        },
        slideDownAndFade: {
          '0%': {
            opacity: 0,
            transform: 'translateY(-3px)',
          },
          '100%': {
            opacity: 1,
            transform: 'translateY(0)',
          },
        },
        slideDown: {
          '0%': {
            height: '0',
          },
          '100%': {
            height: 'var(--radix-accordion-content-height)',
          },
        },
        slideUp: {
          '0%': {
            height: 'var(--radix-accordion-content-height)',
          },
          '100%': {
            height: '0',
          },
        },
      },
      animation: {
        'pulse-spin': 'pulse-spin 1s ease-out infinite',
        // Tooltip
        'slide-down-fade': 'slideDownAndFade 0.2s cubic-bezier(0, 1, 0.3, 1)',
        // Accordion
        'slide-down': 'slideDown 0.3s ease-in-out forwards',
        'slide-up': 'slideUp 0.3s ease-in-out forwards',
      },
    },
  },
  plugins: [require('@headlessui/tailwindcss')],
};
