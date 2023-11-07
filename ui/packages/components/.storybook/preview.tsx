import { withThemeByClassName } from '@storybook/addon-themes';
import type { Preview, ReactRenderer } from '@storybook/react';

import { interTight, robotoMono } from '../src/AppRoot/fonts';
import '../src/AppRoot/globals.css';

const preview: Preview = {
  decorators: [
    (Story) => {
      return (
        <div
          className={`${interTight.variable} ${robotoMono.variable} dark:bg-slate-940 bg-white font-sans`}
        >
          <div id="app" />
          <div id="modals" />
          <Story />
        </div>
      );
    },
    withThemeByClassName<ReactRenderer>({
      themes: {
        light: '',
        dark: 'dark',
      },
      defaultTheme: 'dark',
    }),
  ],
  parameters: {
    nextjs: {
      appDirectory: true,
    },
    actions: { argTypesRegex: '^on[A-Z].*' },
    backgrounds: {
      default: 'dark',
      values: [
        {
          name: 'dark',
          value: '#080D19', // bg-slate-940
        },
        {
          name: 'light',
          value: '#fff',
        },
      ],
    },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/,
      },
    },
    options: {
      storySort: {
        method: 'alphabetical',
      },
    },
  },
};

export default preview;
