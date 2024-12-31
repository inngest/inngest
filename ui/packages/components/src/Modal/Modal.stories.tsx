import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import type { Meta, StoryObj } from '@storybook/react';

import { Modal } from './Modal';

const meta = {
  title: 'Components/Modal',
  component: Modal,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Modal>;

export default meta;

type Story = StoryObj<typeof Modal>;

//@ts-ignore
const ModalWithHooks = (props) => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button label="Open Modal" onClick={() => setIsOpen(true)} />
      <Modal {...props} isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <p className="text-basis p-6">This is the body of the modal</p>
      </Modal>
    </>
  );
};

export const Default: Story = {
  render: () => <ModalWithHooks />,
};

export const WithTitleAndDescription: Story = {
  render: () => <ModalWithHooks title="This is a title" description="This is a description" />,
};
