import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import type { Meta, StoryObj } from '@storybook/react';

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

//@ts-ignore
const SlideOverWithHooks = (props) => {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <>
      <Button label="Open SlideOver" onClick={() => setIsOpen(true)} />
      {isOpen && (
        <SlideOver onClose={() => setIsOpen(false)}>
          <p className="p-6 text-white">This is the body of the SlideOver</p>
        </SlideOver>
      )}
    </>
  );
};

export const Default: Story = {
  render: () => <SlideOverWithHooks />,
};
