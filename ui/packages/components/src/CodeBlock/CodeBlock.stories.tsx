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
    tab: {
      content: '{\n  "customerId": "cus_1234"\n}',
    },
  },
};

export const DefaultWithWrapper: Story = {
  render: () => (
    <CodeBlock.Wrapper>
      <CodeBlock
        header={{ title: 'Output' }}
        tab={{ content: '{\n  "customerId": "cus_1234"\n}' }}
      />
    </CodeBlock.Wrapper>
  ),
};

export const Error: Story = {
  args: {
    header: {
      title: 'Error: Unable to downgrade plan',
      status: 'error',
    },
    tab: {
      content: '{\n  "error": "invalid status code: 500"\n}',
    },
  },
};

export const LongError: Story = {
  args: {
    header: {
      title:
        'Error: Unable to downgrade plan. This is a long message to say that there was an error.',
      status: 'error',
    },
    tab: {
      content: '{\n  "error": "invalid status code: 500"\n}',
    },
  },
};

export const Success: Story = {
  args: {
    header: {
      title: 'Output',
      status: 'success',
    },
    tab: {
      content: '{\n  "customerId": "cus_1234"\n}',
    },
  },
};

export const Actions: Story = {
  args: {
    header: {
      title: 'Output',
    },
    tab: {
      content: '{\n  "customerId": "cus_1234"\n}',
    },

    actions: [
      {
        label: 'Send to Dev Server',
        icon: <IconCloudArrowDown />,
        onClick: () => alert('Sending to dev server...'),
      },
    ],
  },
};
