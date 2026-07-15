import type { Meta, StoryObj } from '@storybook/react';

import { SessionsEmptyState } from './SessionsEmptyState';

const meta = {
  title: 'Components/SessionsEmptyState',
  component: SessionsEmptyState,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof SessionsEmptyState>;

export default meta;

type Story = StoryObj<typeof SessionsEmptyState>;

export const Default: Story = {};
