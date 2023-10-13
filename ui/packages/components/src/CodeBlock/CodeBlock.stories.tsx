import type { Meta, StoryObj } from '@storybook/react';

import CodeBlock from './CodeBlock';

const meta = {
  title: 'Components/CodeBlock',
  component: CodeBlock,
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
} satisfies Meta<typeof CodeBlock>;

export default meta;

type Story = StoryObj<typeof CodeBlock>;

export const Default: Story = {
  args: {
    tabs: [
      {
        label: 'Output',
        content: '{\n  "customerId": "cus_1234"\n}',
      },
    ],
  },
};

export const MultipleTabs: Story = {
  args: {
    tabs: [
      {
        label: 'Output',
        content: '{\n  "customerId": "cus_1234"\n}',
      },
      {
        label: 'Error',
        content: '{\n  "error": "invalid status code: 500"\n}',
      },
    ],
  },
};
