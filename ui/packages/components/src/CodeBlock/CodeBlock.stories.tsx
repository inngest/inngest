import type { Meta, StoryObj } from '@storybook/react';

import { IconCloudArrowDown } from '../icons/CloudArrowDown';
import { CodeBlock } from './CodeBlock';

const meta = {
  title: 'Components/CodeBlock',
  component: CodeBlock,
  decorators: [
    (Story) => (
      <div className="w-[480px]">
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
    header: {
      title: 'Output',
    },
    tabs: [
      {
        label: 'Output',
        content: '{\n  "customerId": "cus_1234"\n}',
      },
    ],
  },
};

export const ErrorCode: Story = {
  args: {
    header: {
      title: 'Error: Unable to downgrade plan',
      status: 'error',
    },
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

export const Actions: Story = {
  args: {
    tabs: [
      {
        label: 'Output',
        content: '{\n  "customerId": "cus_1234"\n}',
      },
    ],
    actions: [
      {
        label: 'Send to Dev Server',
        icon: <IconCloudArrowDown />,
        onClick: () => alert('Sending to dev server...'),
      },
    ],
  },
};
