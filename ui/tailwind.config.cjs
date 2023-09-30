/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  safelist: [
    'text-white',
    'text-slate-100',
    // Primary Button
    'bg-indigo-500',
    'bg-indigo-500/20',
    'bg-indigo-500/80',
    'border-indigo-500',
    'border-indigo-500/80',
    'hover:border-indigo-500',
    'hover:bg-indigo-500/80',
    'text-indigo-500',
    'hover:text-indigo-500/80',
    // Success Button
    'bg-emerald-600',
    'bg-emerald-600/20',
    'bg-emerald-600/80',
    'border-emerald-600',
    'border-emerald-600/80',
    'hover:border-emerald-600',
    'hover:bg-emerald-600/80',
    'text-emerald-600',
    'hover:text-emerald-600/80',
    // Danger Button
    'bg-rose-700',
    'bg-rose-700/20',
    'bg-rose-700/80',
    'border-rose-700',
    'border-rose-700/80',
    'hover:border-rose-700',
    'hover:bg-rose-700/80',
    'text-rose-500',
    'hover:text-rose-500/80',
    // Default Button
    'bg-slate-900',
    'bg-slate-800',
    'bg-slate-800/20',
    'bg-slate-800/80',
    'border-slate-800',
    'border-slate-800/80',
    'hover:border-slate-800',
    'hover:bg-slate-800/80',
    'text-slate-800',
    'hover:text-slate-800/80',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['var(--font-inter)'],
        mono: ["var(--font-roboto-mono)"], 
      },
      colors: {
        slate: {
          950: '#0C1323',
          1000: '#080D19',
        },
      },
      gridTemplateColumns: {
        // Timeline Scroll | Content Frame
        'event-overlay': '340px 1fr',
        app: '1fr',
      },
      gridTemplateRows: {
        // Header | Content Frame
        app: '50px 1fr',
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
        'slide-down-fade': 'slideDownAndFade 0.4s cubic-bezier(0.16, 1, 0.3, 1)',
        // Accordion
        'slide-down': 'slideDown 0.3s ease-in-out forwards',
        'slide-up': 'slideUp 0.3s ease-in-out forwards',
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
