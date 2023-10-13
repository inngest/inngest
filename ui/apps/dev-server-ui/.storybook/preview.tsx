import type { Preview } from '@storybook/react';

import { BaseWrapper } from '@/app/baseWrapper';

const preview: Preview = {
  decorators: [
    (Story) => {
      return (
        <BaseWrapper>
          <Story />
        </BaseWrapper>
      );
    },
  ],
  parameters: {
    actions: { argTypesRegex: '^on[A-Z].*' },
    backgrounds: {
      default: 'dark',
      values: [
        {
          name: 'dark',
          value: '#080D19', // bg-slate-940
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
