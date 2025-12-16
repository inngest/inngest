import type { Meta, StoryObj } from '@storybook/react';

import { CopyButton } from './CopyButton';

const meta = {
  title: 'Components/CopyButton',
  component: CopyButton,
  parameters: {
    layout: 'centered',
  },
  argTypes: {},
} satisfies Meta<typeof CopyButton>;

export default meta;

type Story = StoryObj<typeof CopyButton>;

export const Button: Story = {
  args: {
    code: 'primary',
  },
};

export const Icon: Story = {
  args: {
    code: 'primary',
    iconOnly: true,
  },
};
