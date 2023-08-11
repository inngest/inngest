import { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react';

import Button from './Button';
import SlideOver from './SlideOver';

const meta = {
  title: 'Components/SlideOver',
  component: SlideOver,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof SlideOver>;

export default meta;

type Story = StoryObj<typeof SlideOver>;

const SlideOverWithHooks = (props) => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button label="Open SlideOver" btnAction={() => setIsOpen(true)} />
      {isOpen && (
        <SlideOver onClose={() => setIsOpen(false)}>
          <p className="text-white p-6">This is the body of the SlideOver</p>
        </SlideOver>
      )}
    </>
  );
};

export const Default: Story = {
  render: () => <SlideOverWithHooks />,
};
