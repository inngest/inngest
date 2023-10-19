import { IconFunction } from '@inngest/components/icons/Function';
import type { Meta, StoryObj } from '@storybook/react';

import { SplitButton } from './SplitButton';

const meta = {
  title: 'Components/SplitButton',
  component: SplitButton,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {},
} satisfies Meta<typeof SplitButton>;

export default meta;

const items = [
  {
    label: 'item1',
    icon: <IconFunction />,
    onClick: () => {},
  },
  {
    label: 'item2',
    onClick: () => {},
  },
];

type Story = StoryObj<typeof SplitButton>;

export const Button: Story = {
  args: {
    items: items,
  },
};
