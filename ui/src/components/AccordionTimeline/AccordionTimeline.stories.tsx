import type { Meta, StoryObj } from '@storybook/react';

import AccordionTimeline, { AccordionTimelineItem } from './AccordionTimeline';
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

async function getContent() {
  return "I'm the content";
}

const nodes = [
  {
    id: 'timeline1',
    header: <TimelineItemHeader {...(NoBadge.args as TimelineItemHeaderProps)} />,
    position: 'first',
  },
  {
    id: 'timeline2',
    header: <TimelineItemHeader {...(Default.args as TimelineItemHeaderProps)} />,
    getContent,
    content: <p className="py-6">Content</p>,
  },
  {
    id: 'timeline3',
    header: <TimelineItemHeader {...(Default.args as TimelineItemHeaderProps)} />,
    getContent,
    content: <p className="py-6">Content</p>,
  },
  {
    id: 'timeline4',
    header: <TimelineItemHeader {...(NoBadge.args as TimelineItemHeaderProps)} />,
    position: 'last',
  },
];

export const TimelineList: Story = {
  args: {
    children: nodes.map((node) => {
      const {getContent} = node;

      return (
        <AccordionTimelineItem
          getContent={getContent}
          header={node.header}
          id={node.id}
          key={node.id}
        />
      );
    })
  },
};
