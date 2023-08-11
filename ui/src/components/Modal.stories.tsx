import { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react';

import Button from './Button';
import Modal from './Modal';

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

const ModalWithHooks = (props) => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button label="Open Modal" btnAction={() => setIsOpen(true)} />
      <Modal {...props} isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <p className="text-white p-6">This is the body of the modal</p>
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
