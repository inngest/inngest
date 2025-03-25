import type { Meta, StoryObj } from '@storybook/react';

import { Button } from '../Button';
import EmptyCard from './EmptyCard';

const meta = {
  title: 'Components/Apps/EmptyCard',
  component: EmptyCard,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof EmptyCard>;

export default meta;

type Story = StoryObj<typeof EmptyCard>;

export const Default: Story = {
  args: {
    title: 'No active apps found',
    description:
      'Inngest lets you manage function deployments through apps. Sync your first app to display it here. Need help? Follow our onboarding guide',
    actions: (
      <>
        <Button appearance="outlined" label="Go to docs" />
        <Button kind="primary" label="Sync new app" />
      </>
    ),
  },
};

export const WithoutActions: Story = {
  args: {
    title: 'No active apps found',
    description:
      'Inngest lets you manage function deployments through apps. Sync your first app to display it here. Need help? Follow our onboarding guide',
  },
};
