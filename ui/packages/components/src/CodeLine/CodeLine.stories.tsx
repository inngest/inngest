import type { Meta, StoryObj } from '@storybook/react';

import { CodeLine } from './CodeLine';

const meta = {
  title: 'Components/CodeLine',
  component: CodeLine,
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
} satisfies Meta<typeof CodeLine>;

export default meta;

type Story = StoryObj<typeof CodeLine>;

export const Default: Story = {
  args: {
    code: 'npm install inngest',
  },
};
