import * as AccordionPrimitive from '@radix-ui/react-accordion';
import type { Meta, StoryObj } from '@storybook/react';

import { TimelineNode } from './TimelineNode';

const meta = {
  title: 'Components/TimelineNode',
  component: TimelineNode,
  decorators: [
    (Story) => {
      return (
        <div style={{ width: 600 }}>
          <AccordionPrimitive.Root type="multiple">
            <Story />
          </AccordionPrimitive.Root>
        </div>
      );
    },
  ],
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
} satisfies Meta<typeof TimelineNode>;

export default meta;

type Story = StoryObj<typeof TimelineNode>;

const twoMinutesAgo = new Date(Date.now() - 1000 * 60 * 2);
const oneMinuteAgo = new Date(Date.now() - 1000 * 60);
const now = new Date();
const oneHourAhead = new Date(Date.now() + 1000 * 60 * 60);

const defaultNode = {
  attempt: 0,
  groupID: 'foo',
  isVisible: true,
  name: 'My step',
  scope: 'step',
  status: 'scheduled',
  scheduledAt: twoMinutesAgo,
  endedAt: undefined,
} as const;

export const Cancelled: Story = {
  args: {
    node: {
      ...defaultNode,
      endedAt: now,
      status: 'cancelled',
    },
  },
};

export const Completed: Story = {
  args: {
    node: {
      ...defaultNode,
      endedAt: now,
      status: 'completed',
    },
  },
};

export const CompletedWaitForEvent: Story = {
  args: {
    node: {
      ...defaultNode,
      endedAt: now,
      status: 'completed',
      waitForEventConfig: {
        eventName: 'app/done',
        expression: undefined,
        timeout: oneHourAhead,
      },
      waitForEventResult: {
        eventID: '123',
        timeout: false,
      },
    },
  },
};

export const TimedOutWaitForEvent: Story = {
  args: {
    node: {
      ...defaultNode,
      endedAt: now,
      status: 'completed',
      waitForEventConfig: {
        eventName: 'app/done',
        expression: undefined,
        timeout: now,
      },
      waitForEventResult: {
        eventID: undefined,
        timeout: true,
      },
    },
  },
};

export const Errored: Story = {
  args: {
    node: {
      ...defaultNode,
      status: 'errored',
      outputItemID: '123',
      attempts: {},
    },
    getOutput: async () => 'error code: 524',
  },
};

export const Failed: Story = {
  args: {
    node: {
      ...defaultNode,
      endedAt: now,
      status: 'failed',
    },
  },
};

export const Scheduled: Story = {
  args: {
    node: {
      ...defaultNode,
      status: 'scheduled',
    },
  },
};

export const Sleeping: Story = {
  args: {
    node: {
      ...defaultNode,
      sleepConfig: {
        until: oneHourAhead,
      },
      status: 'sleeping',
    },
  },
};

export const Started: Story = {
  args: {
    node: {
      ...defaultNode,
      startedAt: oneMinuteAgo,
      status: 'started',
    },
  },
};

export const Waiting: Story = {
  args: {
    node: {
      ...defaultNode,
      waitForEventConfig: {
        eventName: 'app/done',
        expression: undefined,
        timeout: oneHourAhead,
      },
      status: 'waiting',
    },
  },
};

export const NoName: Story = {
  args: {
    node: {
      ...defaultNode,
      name: undefined,
    },
  },
};

export const ReallyLongStepName: Story = {
  args: {
    node: {
      ...defaultNode,
      endedAt: now,
      name: 'This is a really long step name that should wrap to the next line',
      status: 'completed',
    },
  },
};
