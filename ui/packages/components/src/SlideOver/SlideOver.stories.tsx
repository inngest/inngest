import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import type { Meta, StoryObj } from '@storybook/react';

import { SlideOver } from './SlideOver';

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
        <SlideOver onClose={() => setIsOpen(false)} size={props.size}>
          <p className="text-basis p-6">This is the body of the SlideOver</p>
        </SlideOver>
      )}
    </>
  );
};

export const Small: Story = {
  render: () => <SlideOverWithHooks size="small" />,
};

export const Large: Story = {
  render: () => <SlideOverWithHooks size="large" />,
};
