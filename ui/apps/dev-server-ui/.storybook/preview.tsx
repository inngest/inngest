import { AppRoot } from '@inngest/components/AppRoot';
import type { Preview } from '@storybook/react';

const preview: Preview = {
  decorators: [
    (Story) => {
      return (
        <AppRoot>
          <Story />
        </AppRoot>
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
          value: '#020202', // carbon-1000
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
