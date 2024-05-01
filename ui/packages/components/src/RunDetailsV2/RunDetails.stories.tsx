import type { Meta, StoryObj } from '@storybook/react';

import { RunDetails } from './RunDetails';

const meta = {
  title: 'Components/RunDetailsV2',
  component: RunDetails,
  parameters: {
    themes: {
      themeOverride: 'light',
    },
  },
} satisfies Meta<typeof RunDetails>;

export default meta;

type Story = StoryObj<typeof RunDetails>;

const app = {
  name: 'My app',
};

const fn = {
  name: 'My Function',
} as const;

const run = {
  id: '01HW3TJKGSYHR2KPW16JJ04M1H',
  output: JSON.stringify({ foo: { bar: 'baz' } }),
};

async function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function getOutput(outputID: string) {
  // Simulate fetch delay
  await sleep(200);

  return JSON.stringify({ foo: { bar: 'baz' } });
}

export const SuccessWithoutSteps: Story = {
  args: {
    app,
    fn,
    getOutput,
    run: {
      ...run,
      trace: {
        attempts: 1,
        endedAt: '2024-04-23T11:26:43.300Z',
        id: '01HW5B7RMP4GSESCWD3VDWE4P0',
        isRoot: true,
        name: 'no-steps-success',
        outputID: 'MDFIVzVDMDVYUVJCWkZKSE02NEJGS01ZS1E=',
        queuedAt: '2024-04-23T11:26:43.260Z',
        startedAt: '2024-04-23T11:26:43.270Z',
        status: 'SUCCEEDED',
        childrenSpans: [
          {
            attempts: 1,
            endedAt: '2024-04-23T11:26:43.300Z',
            id: '01HW5BFZCRYYN2Y33TSMSR8Z0R',
            isRoot: false,
            name: 'Successful Run',
            outputID: null,
            queuedAt: '2024-04-23T11:26:43.260Z',
            startedAt: '2024-04-23T11:26:43.270Z',
            status: 'SUCCEEDED',
            stepInfo: null,
            stepOp: null,
            childrenSpans: [],
          },
        ],
      },
    },
  },
};

export const ErrorWithoutSteps: Story = {
  args: {
    app,
    fn,
    getOutput,
    run: {
      ...run,
      trace: {
        attempts: 1,
        endedAt: '2024-04-23T11:26:43.260Z',
        id: '01HW5B7RMP4GSESCWD3VDWE4P0',
        isRoot: true,
        name: 'no-steps-error',
        outputID: 'MDFIVzVDMUM5RFdTNURQWDlQVjNFWFdERUU=',
        queuedAt: '2024-04-23T11:26:39.260Z',
        startedAt: '2024-04-23T11:26:39.560Z',
        status: 'FAILED',
        childrenSpans: [
          {
            attempts: 5,
            endedAt: '2024-04-23T11:26:43.300Z',
            id: '01HW5BFZCRYYN2Y33TSMSR8Z0R',
            isRoot: false,
            name: 'Failed function',
            outputID: 'MDFIVzVDM0FKRzcxQjc5R00zN0ZCNVdHU1A=',
            queuedAt: '2024-04-23T11:26:39.260Z',
            startedAt: '2024-04-23T11:26:39.560Z',
            status: 'FAILED',
            stepInfo: null,
            stepOp: null,
            childrenSpans: [
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:40.060Z',
                id: '01HW5C7CRFXA0N6BSM58N1F2YZ',
                isRoot: false,
                name: 'Attempt 1',
                outputID: 'MDFIVzVDN1hQSlFGU0YyOEhRQjEyV0RXS0o=',
                queuedAt: '2024-04-23T11:26:39.260Z',
                startedAt: '2024-04-23T11:26:39.360Z',
                status: 'FAILED',
                stepInfo: null,
                stepOp: null,
              },
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:40.860Z',
                id: '01HW5C92PSAC8VEXQ6ECGEMDYE',
                isRoot: false,
                name: 'Attempt 2',
                outputID: 'MDFIVzVDOTJQU0FDOFZFWFE2RUNHRU1EWUU=',
                queuedAt: '2024-04-23T11:26:40.060Z',
                startedAt: '2024-04-23T11:26:40.160Z',
                status: 'FAILED',
                stepInfo: null,
                stepOp: null,
              },
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:41.660Z',
                id: '01HW5C9F7MAKTFY14WF3ZGEV7T',
                isRoot: false,
                name: 'Attempt 3',
                outputID: 'MDFIVzVDOUY3TUFLVEZZMTRXRjNaR0VWN1Q=',
                queuedAt: '2024-04-23T11:26:40.860Z',
                startedAt: '2024-04-23T11:26:40.960Z',
                status: 'FAILED',
                stepInfo: null,
                stepOp: null,
              },
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:42.460Z',
                id: '01HW5C9X9W4P5ZHA7QZ1S4ZQTV',
                isRoot: false,
                name: 'Attempt 4',
                outputID: 'MDFIVzVDOVg5VzRQNVpIQTdRWjFTNFpRVFY=',
                queuedAt: '2024-04-23T11:26:41.660Z',
                startedAt: '2024-04-23T11:26:41.760Z',
                status: 'FAILED',
                stepInfo: null,
                stepOp: null,
              },
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:43.260Z',
                id: '01HW5CA8AKGRPR84PFFB1Y6CWF',
                isRoot: false,
                name: 'Attempt 5',
                outputID: 'MDFIVzVDQThBS0dSUFI4NFBGRkIxWTZDV0Y=',
                queuedAt: '2024-04-23T11:26:42.460Z',
                startedAt: '2024-04-23T11:26:42.560Z',
                status: 'FAILED',
                stepInfo: null,
                stepOp: null,
              },
            ],
          },
        ],
      },
    },
  },
};

export const ParallelRecovery: Story = {
  args: {
    app,
    fn,
    getOutput,
    run: {
      ...run,
      trace: {
        attempts: 1,
        endedAt: '2024-04-23T11:26:43.260Z',
        id: '01HW5CW5E28ZG7GCYSP67ADRND',
        isRoot: true,
        name: 'parallel-recovery',
        outputID: 'MDFIVzVDVzVFMjhaRzdHQ1lTUDY3QURSTkQ=',
        queuedAt: '2024-04-23T11:26:40.260Z',
        startedAt: '2024-04-23T11:26:40.360Z',
        status: 'SUCCEEDED',
        childrenSpans: [
          {
            attempts: 2,
            endedAt: '2024-04-23T11:26:43.260Z',
            id: '01HW5CY6F5Q3C15CY53Z104WTN',
            isRoot: false,
            name: 'Step_id_1',
            outputID: 'MDFIVzVDWTZGNVEzQzE1Q1k1M1oxMDRXVE4=',
            queuedAt: '2024-04-23T11:26:40.460Z',
            startedAt: '2024-04-23T11:26:40.560Z',
            status: 'SUCCEEDED',
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
                status: 'SUCCEEDED',
                stepInfo: null,
                stepOp: 'RUN',
              },
            ],
          },
          {
            attempts: 2,
            endedAt: '2024-04-23T11:26:42.260Z',
            id: '01HW5D84D53Z5KVK4Q35QDNTP4',
            isRoot: false,
            name: 'Step_id_2',
            outputID: 'MDFIVzVEODRENTNaNUtWSzRRMzVRRE5UUDQ=',
            queuedAt: '2024-04-23T11:26:40.460Z',
            startedAt: '2024-04-23T11:26:40.560Z',
            status: 'SUCCEEDED',
            stepInfo: null,
            stepOp: 'RUN',
            childrenSpans: [
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:41.360Z',
                id: '01HW5D9QNGNY5ZRSPQS0RXKM9Q',
                isRoot: false,
                name: 'Attempt 1',
                outputID: 'MDFIVzVEOVFOR05ZNVpSU1BRUzBSWEtNOVE=',
                queuedAt: '2024-04-23T11:26:40.460Z',
                startedAt: '2024-04-23T11:26:40.560Z',
                status: 'FAILED',
                stepInfo: null,
                stepOp: 'RUN',
              },
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:42.260Z',
                id: '01HW5DA4CA3NHAG5QFJ3KGPVZX',
                isRoot: false,
                name: 'Attempt 2',
                outputID: 'MDFIVzVEQTRDQTNOSEFHNVFGSjNLR1BWWlg=',
                queuedAt: '2024-04-23T11:26:41.360Z',
                startedAt: '2024-04-23T11:26:41.460Z',
                status: 'SUCCEEDED',
                stepInfo: null,
                stepOp: 'RUN',
              },
            ],
          },
        ],
      },
    },
  },
};

export const VeryLongStepName: Story = {
  args: {
    app,
    fn,
    getOutput,
    run: {
      ...run,
      trace: {
        attempts: 1,
        endedAt: '2024-04-23T11:26:43.260Z',
        id: '01HW5CW5E28ZG7GCYSP67ADRND',
        isRoot: true,
        name: 'parallel-recovery',
        outputID: 'MDFIVzVDVzVFMjhaRzdHQ1lTUDY3QURSTkQ=',
        queuedAt: '2024-04-23T11:26:40.260Z',
        startedAt: '2024-04-23T11:26:40.360Z',
        status: 'SUCCEEDED',
        childrenSpans: [
          {
            attempts: 2,
            endedAt: '2024-04-23T11:26:43.260Z',
            id: '01HW5CY6F5Q3C15CY53Z104WTN',
            isRoot: false,
            name: 'i-am-a-very-long-step-id-whose-purpose-is-to-test-the-layout',
            outputID: 'MDFIVzVDWTZGNVEzQzE1Q1k1M1oxMDRXVE4=',
            queuedAt: '2024-04-23T11:26:40.460Z',
            startedAt: '2024-04-23T11:26:40.560Z',
            status: 'SUCCEEDED',
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
                status: 'SUCCEEDED',
                stepInfo: null,
                stepOp: 'RUN',
              },
            ],
          },
          {
            attempts: 2,
            endedAt: '2024-04-23T11:26:42.260Z',
            id: '01HW5D84D53Z5KVK4Q35QDNTP4',
            isRoot: false,
            name: 'short-boi',
            outputID: 'MDFIVzVEODRENTNaNUtWSzRRMzVRRE5UUDQ=',
            queuedAt: '2024-04-23T11:26:40.460Z',
            startedAt: '2024-04-23T11:26:40.560Z',
            status: 'SUCCEEDED',
            stepInfo: null,
            stepOp: 'RUN',
            childrenSpans: [
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:41.360Z',
                id: '01HW5D9QNGNY5ZRSPQS0RXKM9Q',
                isRoot: false,
                name: 'Attempt 1',
                outputID: 'MDFIVzVEOVFOR05ZNVpSU1BRUzBSWEtNOVE=',
                queuedAt: '2024-04-23T11:26:40.460Z',
                startedAt: '2024-04-23T11:26:40.560Z',
                status: 'FAILED',
                stepInfo: null,
                stepOp: 'RUN',
              },
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:42.260Z',
                id: '01HW5DA4CA3NHAG5QFJ3KGPVZX',
                isRoot: false,
                name: 'Attempt 2',
                outputID: 'MDFIVzVEQTRDQTNOSEFHNVFGSjNLR1BWWlg=',
                queuedAt: '2024-04-23T11:26:41.360Z',
                startedAt: '2024-04-23T11:26:41.460Z',
                status: 'SUCCEEDED',
                stepInfo: null,
                stepOp: 'RUN',
              },
            ],
          },
        ],
      },
    },
  },
};

export const LongSleep: Story = {
  args: {
    app,
    fn,
    getOutput,
    run: {
      ...run,
      trace: {
        attempts: 1,
        endedAt: '2024-04-30T11:26:43.260Z',
        id: '01HW5CW5E28ZG7GCYSP67ADRND',
        isRoot: true,
        name: 'my-fn',
        outputID: 'MDFIVzVDVzVFMjhaRzdHQ1lTUDY3QURSTkQ=',
        queuedAt: '2024-04-23T11:26:40.260Z',
        startedAt: '2024-04-23T11:26:40.560Z',
        status: 'SUCCEEDED',
        childrenSpans: [
          {
            attempts: 1,
            endedAt: '2024-04-23T11:26:43.260Z',
            id: '01HW5CY6F5Q3C15CY53Z104WTN',
            isRoot: false,
            name: 'step-1',
            outputID: 'MDFIVzVDWTZGNVEzQzE1Q1k1M1oxMDRXVE4=',
            queuedAt: '2024-04-23T11:26:40.460Z',
            startedAt: '2024-04-23T11:26:40.560Z',
            status: 'SUCCEEDED',
            stepInfo: null,
            stepOp: 'RUN',
            childrenSpans: [
              {
                attempts: 1,
                endedAt: '2024-04-23T11:26:43.260Z',
                id: '01HW5D23MY4MCX421M0RJ7W3AJ',
                isRoot: false,
                name: 'Attempt 1',
                outputID: 'MDFIVzVEMjNNWTRNQ1g0MjFNMFJKN1czQUo=',
                queuedAt: '2024-04-23T11:26:40.460Z',
                startedAt: '2024-04-23T11:26:40.560Z',
                status: 'SUCCEEDED',
                stepInfo: null,
                stepOp: 'RUN',
              },
            ],
          },
          {
            attempts: 1,
            endedAt: '2024-04-30T11:26:43.260Z',
            id: '01HW5CY6F5Q3C15CY53Z104WTN',
            isRoot: false,
            name: 'long-sleep',
            outputID: 'MDFIVzVDWTZGNVEzQzE1Q1k1M1oxMDRXVE4=',
            queuedAt: '2024-04-23T11:26:40.460Z',
            startedAt: '2024-04-23T11:26:40.560Z',
            status: 'SUCCEEDED',
            stepInfo: null,
            stepOp: 'RUN',
            childrenSpans: [
              {
                attempts: 1,
                endedAt: '2024-04-30T11:26:43.260Z',
                id: '01HW5D23MY4MCX421M0RJ7W3AJ',
                isRoot: false,
                name: 'Attempt 1',
                outputID: 'MDFIVzVEMjNNWTRNQ1g0MjFNMFJKN1czQUo=',
                queuedAt: '2024-04-23T11:26:40.460Z',
                startedAt: '2024-04-23T11:26:40.560Z',
                status: 'SUCCEEDED',
                stepInfo: null,
                stepOp: 'RUN',
              },
            ],
          },
        ],
      },
    },
  },
};
