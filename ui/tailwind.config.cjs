/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        slate: {
          950: '#0C1323',
          1000: '#080D19',
        },
      },
      gridTemplateColumns: {
        // Sidebar | Timeline Scroll | Content Frame
        'app-sm': '260px 1fr',
        app: '340px 1fr',
        'app-desktop': '420px 1fr',
        'event-overlay': '340px 1fr',
        docs: '1fr',
      },
      gridTemplateRows: {
        // Header | Action Bar | Content Frame
        app: '50px 50px 1fr',
        docs: '50px 1fr',
        'event-overlay': '120px 1fr',
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
      },
      animation: {
        'pulse-spin': 'pulse-spin 1s ease-out infinite',
      },
    },
    fontSize: {
      '4xs': '0.625rem', // 10px
      '3xs': '0.6875rem', // 11px
      '2xs': '0.75rem', // 12px
      xs: '0.8125rem', // 13px
      sm: '0.875rem', // 14px
      base: '1rem', // 16px
      lg: '1.125rem', // 18px
      xl: '1.25rem', // 20px
      '2xl': '1.5rem', // 24px
      '3xl': '1.875rem', // 30px
      '4xl': '2.25rem', // 36px
      '5xl': '3rem', // 48px
      '6xl': '3.75rem', // 60p
    },
    borderRadius: {
      none: '0',
      sm: '4px',
      md: '6px',
      DEFAULT: '8px',
      lg: '10px',
      full: '9999px',
    },
  },
  plugins: [
    require('@headlessui/tailwindcss'),
    function ({ addUtilities, theme }) {
      const iconSizeUtilities = {};
      Object.keys(theme('fontSize')).forEach((size) => {
        const value = theme('fontSize')[size];
        iconSizeUtilities[`.icon-${size}`] = {
          width: value,
          height: value,
        };
      });
      addUtilities(iconSizeUtilities, ['responsive', 'hover']);
    },
  ],
};
