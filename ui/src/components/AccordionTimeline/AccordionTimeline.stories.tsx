import type { Meta, StoryObj } from '@storybook/react';

import AccordionTimeline from './AccordionTimeline';
import TimelineItemHeader, { type TimelineItemHeaderProps } from './TimelineItemHeader';
import { Default, NoBadge } from './TimelineItemHeader.stories';

const meta = {
  title: 'Components/AccordionTimeline',
  component: AccordionTimeline,
  parameters: {
    layout: 'centered',
  },
  decorators: [
    (Story) => {
      return (
        <div style={{ width: 600 }}>
          <Story />
        </div>
      );
    },
  ],
  tags: ['autodocs'],
} satisfies Meta<typeof AccordionTimeline>;

export default meta;

type Story = StoryObj<typeof AccordionTimeline>;

export const TimelineList: Story = {
  args: {
    timelineItems: [
      {
        id: 'timeline1',
        header: <TimelineItemHeader {...(NoBadge.args as TimelineItemHeaderProps)} />,
        expandable: true,
        content: <p className="py-6">Content</p>,
        position: 'first',
      },
      {
        id: 'timeline2',
        header: <TimelineItemHeader {...(Default.args as TimelineItemHeaderProps)} />,
        expandable: true,
        content: <p className="py-6">Content</p>,
      },
      {
        id: 'timeline3',
        header: <TimelineItemHeader {...(NoBadge.args as TimelineItemHeaderProps)} />,
        expandable: true,
        content: <p className="py-6">Content</p>,
        position: 'last',
      },
    ],
  },
};
