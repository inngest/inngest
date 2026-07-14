import type { Meta, StoryObj } from '@storybook/react';

import { ExperimentsEmptyState } from './ExperimentsEmptyState';

const meta = {
  title: 'Components/ExperimentsEmptyState',
  component: ExperimentsEmptyState,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof ExperimentsEmptyState>;

export default meta;

type Story = StoryObj<typeof ExperimentsEmptyState>;

export const Default: Story = {};
