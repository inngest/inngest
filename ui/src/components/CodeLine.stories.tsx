import type { Meta, StoryObj } from '@storybook/react';

import CodeLine from './CodeLine';

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

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    className: 'p-4',
    code: 'npm install inngest',
  },
};
