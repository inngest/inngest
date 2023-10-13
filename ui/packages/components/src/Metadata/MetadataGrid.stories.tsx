import type { Meta, StoryObj } from '@storybook/react';

import { MetadataGrid } from './MetadataGrid';
import { type MetadataItemProps } from './MetadataItem';

const MetadataArray: MetadataItemProps[] = [
  {
    label: 'Attempt Started',
    value: '27/07/2023, 21:45:38',
    size: 'large',
  },
  {
    label: 'Attempt Started',
    value: '27/07/2023, 21:45:38',
  },
  {
    label: 'Attempt Started',
    value: '27/07/2023, 21:45:38',
  },
  {
    label: 'Attempt Started',
    value: '27/07/2023, 21:45:38',
    size: 'large',
    tooltip: 'This is the timestamp of the attempt',
  },
  {
    label: 'Attempt Started',
    value: '27/07/2023, 21:45:38',
  },
  {
    label: 'Attempt Started',
    value: '27/07/2023, 21:45:38',
  },
];

const meta = {
  title: 'Components/MetadataGrid',
  component: MetadataGrid,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof MetadataGrid>;

export default meta;

type Story = StoryObj<typeof MetadataGrid>;

export const OneItemGrid: Story = {
  args: {
    metadataItems: MetadataArray.slice(0, 1),
  },
};

export const OneLargeAndOneRegularItemGrid: Story = {
  args: {
    metadataItems: MetadataArray.slice(0, 2),
  },
};

export const TwoItemsGrid: Story = {
  args: {
    metadataItems: MetadataArray.slice(1, 3),
  },
};

export const TwoRowsGrid: Story = {
  args: {
    metadataItems: MetadataArray.slice(0, 4),
  },
};

export const ThreeRowsGrid: Story = {
  args: {
    metadataItems: MetadataArray,
  },
};
