import type { Meta, StoryObj } from '@storybook/react';

import { Trace } from './Trace';

const meta = {
  title: 'Components/Trace',
  component: Trace,
} satisfies Meta<typeof Trace>;

export default meta;

async function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function getOutput(outputID: string) {
  // Simulate fetch delay
  await sleep(200);

  return JSON.stringify({ foo: { bar: 'baz' } });
}

type Story = StoryObj<typeof Trace>;

export const Retry: Story = {
  args: {
    depth: 0,
    getResult: getOutput,
    trace: {
      attempts: 2,
      endedAt: '2024-04-23T11:26:43.260Z',
      id: '01HW5CY6F5Q3C15CY53Z104WTN',
      isRoot: false,
      name: 'Step_id_1',
      outputID: 'MDFIVzVDWTZGNVEzQzE1Q1k1M1oxMDRXVE4=',
      queuedAt: '2024-04-23T11:26:40.460Z',
      startedAt: '2024-04-23T11:26:40.560Z',
      status: 'COMPLETED',
      stepInfo: null,
      stepOp: 'RUN',
      childrenSpans: [
        {
          attempts: 1,
          endedAt: '2024-04-23T11:26:42.060Z',
          id: '01HW5D1GRC1AAYJXSNFPC16BHM',
          isRoot: false,
          name: 'Attempt 1',
          outputID: 'MDFIVzVEMUdSQzFBQVlKWFNORlBDMTZCSE0=',
          queuedAt: '2024-04-23T11:26:40.460Z',
          startedAt: '2024-04-23T11:26:40.560Z',
          status: 'FAILED',
          stepInfo: null,
          stepOp: 'RUN',
        },
        {
          attempts: 1,
          endedAt: '2024-04-23T11:26:43.260Z',
          id: '01HW5D23MY4MCX421M0RJ7W3AJ',
          isRoot: false,
          name: 'Attempt 2',
          outputID: 'MDFIVzVEMjNNWTRNQ1g0MjFNMFJKN1czQUo=',
          queuedAt: '2024-04-23T11:26:42.060Z',
          startedAt: '2024-04-23T11:26:42.160Z',
          status: 'COMPLETED',
          stepInfo: null,
          stepOp: 'RUN',
        },
      ],
    },
  },
};
