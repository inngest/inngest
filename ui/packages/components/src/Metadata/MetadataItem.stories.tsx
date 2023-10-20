import type { Meta, StoryObj } from '@storybook/react';

import { MetadataItem } from './MetadataItem';

const meta = {
  title: 'Components/MetadataItem',
  component: MetadataItem,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof MetadataItem>;

export default meta;

type Story = StoryObj<typeof MetadataItem>;

export const Default: Story = {
  args: {
    label: 'Attempt Started',
    value: '27/07/2023, 21:45:38',
  },
};

export const WithTooltip: Story = {
  args: {
    label: 'Attempt Started',
    value: '27/07/2023, 21:45:38',
    tooltip: 'This is the timestamp of the attempt',
  },
};
