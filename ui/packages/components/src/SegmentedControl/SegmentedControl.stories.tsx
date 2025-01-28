import { RiMoonClearFill, RiSunLine, RiWindow2Line } from '@remixicon/react';
import type { Meta, StoryObj } from '@storybook/react';

import SegmentedControl from './SegmentedControl';

const meta = {
  title: 'Components/SegmentedControl',
  component: SegmentedControl,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof SegmentedControl>;

export default meta;

type Story = StoryObj<typeof SegmentedControl>;

export const Default: Story = {
  render: () => (
    <SegmentedControl defaultValue="first">
      <SegmentedControl.Button value="first">First</SegmentedControl.Button>
      <SegmentedControl.Button value="second">Second</SegmentedControl.Button>
      <SegmentedControl.Button value="third">Third</SegmentedControl.Button>
    </SegmentedControl>
  ),
};

export const WithIcons: Story = {
  render: () => (
    <SegmentedControl defaultValue="first">
      <SegmentedControl.Button value="first" icon={<RiSunLine />} iconSide="left">
        First
      </SegmentedControl.Button>
      <SegmentedControl.Button value="second" icon={<RiMoonClearFill />} iconSide="left">
        Second
      </SegmentedControl.Button>
      <SegmentedControl.Button value="third" icon={<RiMoonClearFill />} iconSide="left">
        Third
      </SegmentedControl.Button>
    </SegmentedControl>
  ),
};

export const OnlyIcons: Story = {
  render: () => (
    <SegmentedControl defaultValue="light">
      <SegmentedControl.Button value="light" icon={<RiSunLine />} />
      <SegmentedControl.Button value="dark" icon={<RiMoonClearFill />} />
      <SegmentedControl.Button value="system" icon={<RiWindow2Line className="rotate-180" />} />
    </SegmentedControl>
  ),
};
