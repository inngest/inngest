import fs from 'fs/promises';
import path from 'path';
import { expect, test } from 'vitest';

import { HistoryParser } from './historyParser';
import type { HistoryNode } from './types';

async function loadHistory(filename: string): Promise<HistoryParser> {
  const raw = JSON.parse(await fs.readFile(path.join(__dirname, `testData/${filename}`), 'utf8'));
  return new HistoryParser(raw);
}

const baseRunStartNode = {
  attempt: 0,
  attempts: {},
  endedAt: expect.any(Date),
  groupID: expect.any(String),
  name: undefined,
  outputItemID: undefined,
  scheduledAt: expect.any(Date),
  scope: 'function',
  sleepConfig: undefined,
  status: 'started',
  startedAt: expect.any(Date),
  url: undefined,
  waitForEventConfig: undefined,
  waitForEventResult: undefined,
} as const;

const baseStepNode = {
  attempt: 0,
  attempts: {},
  endedAt: expect.any(Date),
  groupID: expect.any(String),
  outputItemID: expect.any(String),
  scheduledAt: expect.any(Date),
  scope: 'step',
  sleepConfig: undefined,
  startedAt: expect.any(Date),
  url: 'http://localhost:3939/api/inngest',
  waitForEventConfig: undefined,
  waitForEventResult: undefined,
} as const;

// We default to 2 attempts but the last one won't appear in a root-level node's
// `attempts` field. That's because the top-level node is the latest attempt.
const baseAttempts = {
  '0': {
    ...baseStepNode,
    outputItemID: expect.any(String),
    status: 'errored',
  },
  '1': {
    ...baseStepNode,
    attempt: 1,
    outputItemID: expect.any(String),
    status: 'errored',
  },
  '2': {
    ...baseStepNode,
    attempt: 2,
    outputItemID: expect.any(String),
    status: 'failed',
  },
} as const;

const baseRunEndNode = {
  attempt: 0,
  attempts: {},
  endedAt: expect.any(Date),
  groupID: expect.any(String),
  name: undefined,
  outputItemID: expect.any(String),
  scheduledAt: expect.any(Date),
  scope: 'function',
  sleepConfig: undefined,
  startedAt: expect.any(Date),
  url: 'http://localhost:3939/api/inngest',
  waitForEventConfig: undefined,
  waitForEventResult: undefined,
} as const;

test('cancels', async () => {
  const history = await loadHistory('cancels.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseStepNode,
      status: 'cancelled',
      name: '1m',
      outputItemID: expect.any(String),
      sleepConfig: {
        until: expect.any(Date),
      },
    },
    {
      ...baseRunEndNode,
      outputItemID: undefined,
      startedAt: undefined,
      status: 'cancelled',
      url: undefined,
    },
  ];

  expect(history.getGroups({ sort: true })).toEqual(expectation);
  expect(history.cancellation).toEqual({
    eventID: expect.any(String),
    expression: null,
    userID: null,
  });
});

test('fails without steps', async () => {
  const history = await loadHistory('failsWithoutSteps.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseStepNode,
      attempt: 2,
      attempts: baseAttempts,
      status: 'failed',
    },
    {
      ...baseRunEndNode,
      attempt: 2,
      status: 'failed',
    },
  ];

  expect(history.getGroups({ sort: true })).toEqual(expectation);
});

test('fails with preceding step', async () => {
  const history = await loadHistory('failsWithPrecedingStep.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseStepNode,
      status: 'completed',
      name: 'First step',
    },
    {
      ...baseStepNode,
      attempt: 2,
      attempts: baseAttempts,
      status: 'failed',
    },
    {
      ...baseRunEndNode,
      attempt: 2,
      status: 'failed',
    },
  ];

  expect(history.getGroups({ sort: true })[2]).toEqual(expectation[2]);
});

test('no steps', async () => {
  const history = await loadHistory('noSteps.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseRunEndNode,
      status: 'completed',
    },
  ];

  expect(history.getGroups({ sort: true })).toEqual(expectation);
});

test('parallel steps', async () => {
  const history = await loadHistory('parallelSteps.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseStepNode,
      status: 'completed',
      name: 'a',
    },
    {
      ...baseStepNode,
      status: 'completed',
    },
    {
      ...baseStepNode,
      status: 'completed',
      name: 'b2',
    },
    {
      ...baseStepNode,
      status: 'completed',
      name: 'b1',
    },
    {
      ...baseStepNode,
      status: 'completed',
    },
    {
      ...baseRunEndNode,
      status: 'completed',
    },
  ];

  expect(history.getGroups({ sort: true })).toEqual(expectation);
});

test('sleeps', async () => {
  const history = await loadHistory('sleeps.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseStepNode,
      status: 'completed',
      name: '10s',
      sleepConfig: {
        until: expect.any(Date),
      },
    },
    {
      ...baseRunEndNode,
      status: 'completed',
    },
  ];

  expect(history.getGroups({ sort: true })).toEqual(expectation);
});

test('succeeds with 2 steps', async () => {
  const history = await loadHistory('succeedsWith2Steps.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseStepNode,
      status: 'completed',
      name: 'First step',
    },
    {
      ...baseStepNode,
      status: 'completed',
      name: 'Second step',
    },
    {
      ...baseRunEndNode,
      status: 'completed',
    },
  ];

  expect(history.getGroups({ sort: true }).slice(0, 1)).toEqual(expectation.slice(0, 1));
});

test('times out waiting for events', async () => {
  const history = await loadHistory('timesOutWaitingForEvent.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseStepNode,
      status: 'completed',
      waitForEventResult: {
        eventID: undefined,
        timeout: true,
      },
      waitForEventConfig: {
        eventName: 'bar',
        expression: undefined,
        timeout: expect.any(Date),
      },
    },
    {
      ...baseRunEndNode,
      status: 'completed',
    },
  ];

  expect(history.getGroups({ sort: true })).toEqual(expectation);
});

test('waits for event', async () => {
  const history = await loadHistory('waitsForEvent.json');

  const expectation: HistoryNode[] = [
    baseRunStartNode,
    {
      ...baseStepNode,
      status: 'completed',
      waitForEventResult: {
        eventID: expect.any(String),
        timeout: false,
      },
      waitForEventConfig: {
        eventName: 'bar',
        expression: undefined,
        timeout: expect.any(Date),
      },
    },
    {
      ...baseRunEndNode,
      status: 'completed',
    },
  ];

  expect(history.getGroups({ sort: true })).toEqual(expectation);
});
