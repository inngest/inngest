import type { Meta, StoryObj } from '@storybook/react';

import { CodeKey } from './CodeKey';

const meta = {
  title: 'Components/CodeKey',
  component: CodeKey,
  decorators: [
    (Story) => (
      <div className="w-80">
        <Story />
      </div>
    ),
  ],
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof CodeKey>;

export default meta;

type Story = StoryObj<typeof CodeKey>;

export const Default: Story = {
  args: {
    fullKey: 'FNb31q4iFNb31q4iFNb31q4iFNb31q4iFNb31q4iFNb31q4i',
    maskedKey: 'FNb31q4i...',
    label: 'Signing Key',
  },
};
