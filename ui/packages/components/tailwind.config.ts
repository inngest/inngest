import type { Config } from 'tailwindcss';
import defaultTheme from 'tailwindcss/defaultTheme';

export default {
  content: ['./src/**/*.{ts,tsx,mdx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['var(--font-circular)', ...defaultTheme.fontFamily.sans],
        mono: ['var(--font-circular-mono)', ...defaultTheme.fontFamily.mono],
      },
      colors: {
        slate: {
          910: '#0C1323',
          940: '#080D19',
        },
        primary: {
          '2xSubtle': 'rgb(var(--color-primary-2xSubtle) / <alpha-value>)',
          xSubtle: 'rgb(var(--color-primary-xSubtle) / <alpha-value>)',
          subtle: 'rgb(var(--color-primary-subtle) / <alpha-value>)',
          moderate: 'rgb(var(--color-primary-moderate) / <alpha-value>)',
          intense: 'rgb(var(--color-primary-intense) / <alpha-value>)',
          xIntense: 'rgb(var(--color-primary-xIntense) / <alpha-value>)',
          '2xIntense': 'rgb(var(--color-primary-2xIntense) / <alpha-value>)',
        },
        secondary: {
          '4xSubtle': 'rgb(var(--color-secondary-4xSubtle) / <alpha-value>)',
          '3xSubtle': 'rgb(var(--color-secondary-3xSubtle) / <alpha-value>)',
          '2xSubtle': 'rgb(var(--color-secondary-2xSubtle) / <alpha-value>)',
          xSubtle: 'rgb(var(--color-secondary-xSubtle) / <alpha-value>)',
          subtle: 'rgb(var(--color-secondary-subtle) / <alpha-value>)',
          moderate: 'rgb(var(--color-secondary-moderate) / <alpha-value>)',
          intense: 'rgb(var(--color-secondary-intense) / <alpha-value>)',
          xIntense: 'rgb(var(--color-secondary-xIntense) / <alpha-value>)',
          '2xIntense': 'rgb(var(--color-secondary-2xIntense) / <alpha-value>)',
        },
        tertiary: {
          '2xSubtle': 'rgb(var(--color-tertiary-2xSubtle) / <alpha-value>)',
          xSubtle: 'rgb(var(--color-tertiary-xSubtle) / <alpha-value>)',
          subtle: 'rgb(var(--color-tertiary-subtle) / <alpha-value>)',
          moderate: 'rgb(var(--color-tertiary-moderate) / <alpha-value>)',
          intense: 'rgb(var(--color-tertiary-intense) / <alpha-value>)',
          xIntense: 'rgb(var(--color-tertiary-xIntense) / <alpha-value>)',
          '2xIntense': 'rgb(var(--color-tertiary-2xIntense) / <alpha-value>)',
        },
        quaternary: {
          coolxSubtle: 'rgb(var(--color-quaternary-cool-xSubtle) / <alpha-value>)',
          coolModerate: 'rgb(var(--color-quaternary-cool-moderate) / <alpha-value>)',
          coolxIntense: 'rgb(var(--color-quaternary-cool-xIntense) / <alpha-value>)',
        },
        accent: {
          '2xSubtle': 'rgb(var(--color-accent-2xSubtle) / <alpha-value>)',
          xSubtle: 'rgb(var(--color-accent-xSubtle) / <alpha-value>)',
          subtle: 'rgb(var(--color-accent-subtle) / <alpha-value>)',
          moderate: 'rgb(var(--color-accent-moderate) / <alpha-value>)',
          intense: 'rgb(var(--color-accent-intense) / <alpha-value>)',
          xIntense: 'rgb(var(--color-accent-xIntense) / <alpha-value>)',
          '2xIntense': 'rgb(var(--color-accent-2xIntense) / <alpha-value>)',
        },
        status: {
          failed: 'rgb(var(--color-tertiary-subtle) / <alpha-value>)',
          failedText: 'rgb(var(--color-tertiary-intense) / <alpha-value>)',
          running: 'rgb(var(--color-secondary-subtle) / <alpha-value>)',
          runningSubtle: 'rgb(var(--color-secondary-2xSubtle) / <alpha-value>)',
          runningText: 'rgb(var(--color-secondary-intense) / <alpha-value>)',
          queued: 'rgb(var(--color-quaternary-cool-moderate) / <alpha-value>)',
          queuedSubtle: 'rgb(var(--color-quaternary-cool-xSubtle) / <alpha-value>)',
          queuedText: 'rgb(var(--color-quaternary-cool-xIntense) / <alpha-value>)',
          completed: 'rgb(var(--color-primary-subtle) / <alpha-value>)',
          completedText: 'rgb(var(--color-primary-intense) / <alpha-value>)',
          cancelled: 'rgb(var(--color-foreground-cancelled) / <alpha-value>)',
          cancelledText: 'rgb(var(--color-foreground-subtle) / <alpha-value>)',
          paused: 'rgb(var(--color-foreground-paused) / <alpha-value>)',
          pausedText: 'rgb(var(--color-foreground-subtle) / <alpha-value>)',
        },
      },
      borderColor: {
        subtle: 'rgb(var(--color-border-subtle) / <alpha-value>)',
        muted: 'rgb(var(--color-border-muted) / <alpha-value>)',
        contrast: 'rgb(var(--color-border-contrast) / <alpha-value>)',
        disabled: 'rgb(var(--color-border-disabled) / <alpha-value>)',
        success: 'rgb(var(--color-border-success) / <alpha-value>)',
        error: 'rgb(var(--color-border-error) / <alpha-value>)',
        warning: 'rgb(var(--color-border-warning) / <alpha-value>)',
        info: 'rgb(var(--color-border-info) / <alpha-value>)',
      },
      backgroundColor: {
        canvasBase: 'rgb(var(--color-background-canvas-base) / <alpha-value>)',
        canvasSubtle: 'rgb(var(--color-background-canvas-subtle) / <alpha-value>)',
        canvasMuted: 'rgb(var(--color-background-canvas-muted) / <alpha-value>)',
        surfaceBase: 'rgb(var(--color-background-surface-base) / <alpha-value>)',
        surfaceSubtle: 'rgb(var(--color-background-surface-subtle) / <alpha-value>)',
        surfaceMuted: 'rgb(var(--color-background-surface-muted) / <alpha-value>)',
        disabled: 'rgb(var(--color-background-disabled) / <alpha-value>)',
        contrast: 'rgb(var(--color-background-contrast) / <alpha-value>)',
        success: 'rgb(var(--color-background-success) / <alpha-value>)',
        successContrast: 'rgb(var(--color-background-successContrast) / <alpha-value>)',
        error: 'rgb(var(--color-background-error) / <alpha-value>)',
        errorContrast: 'rgb(var(--color-background-errorContrast) / <alpha-value>)',
        warning: 'rgb(var(--color-background-warning) / <alpha-value>)',
        warningContrast: 'rgb(var(--color-background-warningContrast) / <alpha-value>)',
        info: 'rgb(var(--color-background-info) / <alpha-value>)',
        infoContrast: 'rgb(var(--color-background-infoContrast) / <alpha-value>)',
        codeEditor: 'rgb(var(--color-background-codeEditor) / <alpha-value>)',
        btnPrimary: 'rgb(var(--color-background-btn-primary) / <alpha-value>)',
        btnPrimaryHover: 'rgb(var(--color-background-btn-primaryHover) / <alpha-value>)',
        btnPrimaryPressed: 'rgb(var(--color-background-btn-primaryPressed) / <alpha-value>)',
        btnPrimaryDisabled: 'rgb(var(--color-background-btn-primaryDisabled) / <alpha-value>)',
        btnDanger: 'rgb(var(--color-background-btn-danger) / <alpha-value>)',
        btnDangerHover: 'rgb(var(--color-background-btn-dangerHover) / <alpha-value>)',
        btnDangerPressed: 'rgb(var(--color-background-btn-dangerPressed) / <alpha-value>)',
        btnDangerDisabled: 'rgb(var(--color-background-btn-dangerDisabled) / <alpha-value>)',
      },
      textColor: {
        basis: 'rgb(var(--color-foreground-base) / <alpha-value>)',
        subtle: 'rgb(var(--color-foreground-subtle) / <alpha-value>)',
        muted: 'rgb(var(--color-foreground-muted) / <alpha-value>)',
        onContrast: 'rgb(var(--color-foreground-onContrast) / <alpha-value>)',
        alwaysWhite: 'rgb(var(--color-foreground-alwaysWhite) / <alpha-value>)',
        alwaysBlack: 'rgb(var(--color-foreground-alwaysBlack) / <alpha-value>)',
        disabled: 'rgb(var(--color-foreground-disabled) / <alpha-value>)',
        link: 'rgb(var(--color-foreground-link) / <alpha-value>)',
        success: 'rgb(var(--color-foreground-success) / <alpha-value>)',
        error: 'rgb(var(--color-foreground-error) / <alpha-value>)',
        warning: 'rgb(var(--color-foreground-warning) / <alpha-value>)',
        info: 'rgb(var(--color-foreground-info) / <alpha-value>)',
        btnPrimary: 'rgb(var(--color-foreground-btn-primary) / <alpha-value>)',
        btnPrimaryDisabled: 'rgb(var(--color-foreground-btn-primaryDisabled) / <alpha-value>)',
        btnDanger: 'rgb(var(--color-foreground-btn-danger) / <alpha-value>)',
        btnDangerDisabled: 'rgb(var(--color-foreground-btn-dangerDisabled) / <alpha-value>)',
      },
      textDecorationColor: {
        link: 'rgb(var(--color-foreground-link) / <alpha-value>)',
      },
      fill: {
        // temporary tooltip token
        tooltipArrow: 'rgb(var(--color-background-canvas-base) / <alpha-value>)',
        onContrast: 'rgb(var(--color-foreground-onContrast) / <alpha-value>)',
        subtle: 'rgb(var(--color-foreground-subtle) / <alpha-value>)',
        alwaysWhite: 'rgb(var(--color-foreground-alwaysWhite) / <alpha-value>)',
        btnPrimary: 'rgb(var(--color-foreground-btn-primary) / <alpha-value>)',
        btnDanger: 'rgb(var(--color-foreground-btn-danger) / <alpha-value>)',
      },
      gridTemplateColumns: {
        dashboard: '1fr 1fr 1fr 432px',
      },
      boxShadowColor: {
        // temporary tooltip token
        tooltip: 'rgb(var(--color-background-canvas-muted) / <alpha-value>)',
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
      },
      animation: {
        'slide-down-and-fade': 'slide-down-and-fade 400ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-left-and-fade': 'slide-left-and-fade 400ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-up-and-fade': 'slide-up-and-fade 400ms cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-right-and-fade': 'slide-right-and-fade 400ms cubic-bezier(0.16, 1, 0.3, 1)',
      },
    },
  },
  plugins: [require('@headlessui/tailwindcss')],
} satisfies Config;
