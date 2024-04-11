import { TooltipProvider } from '@inngest/components/Tooltip';
import { withThemeByClassName } from '@storybook/addon-themes';
import type { Preview, ReactRenderer } from '@storybook/react';

import { interTight, robotoMono } from '../src/AppRoot/fonts';
import '../src/AppRoot/globals.css';

const preview: Preview = {
  decorators: [
    (Story) => {
      return (
        <TooltipProvider>
          <div className={`${interTight.variable} ${robotoMono.variable} font-sans`}>
            <div id="app" />
            <div id="modals" />
            <Story />
          </div>
        </TooltipProvider>
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
