import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import type { Meta, StoryObj } from '@storybook/react';

import { AlertModal } from './AlertModal';

const meta = {
  title: 'Components/AlertModal',
  component: AlertModal,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof AlertModal>;

export default meta;

type Story = StoryObj<typeof AlertModal>;

//@ts-ignore
const ModalWithHooks = (props) => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button label="Delete" onClick={() => setIsOpen(true)} />
      <AlertModal {...props} isOpen={isOpen} onClose={() => setIsOpen(false)} onSubmit={() => {}} />
    </>
  );
};

export const Default: Story = {
  render: () => <ModalWithHooks />,
};

export const WithTitleAndDescription: Story = {
  render: () => (
    <ModalWithHooks
      title="Are you sure you want to delete the account?"
      description="This action cannot be undone. This will permanently delete your account."
    />
  ),
};
