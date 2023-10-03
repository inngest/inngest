import fs from 'fs/promises';
import path from 'path';
import { expect, test } from 'vitest';

import { HistoryParser } from './historyParser.js';
import type { HistoryNode } from './types.js';

async function loadHistory(filename: string) {
  const raw = JSON.parse(await fs.readFile(path.join(__dirname, `testData/${filename}`), 'utf8'));

  return Object.values(new HistoryParser(raw).history).sort((a, b) => {
    return a.scheduledAt.getTime() - b.scheduledAt.getTime();
  });
}

const baseStepNode = {
  attempt: 0,
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

const baseRunEndNode = {
  attempt: 0,
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

  expect(history).toEqual(expectation);
});

test('fails without steps', async () => {
  const history = await loadHistory('failsWithoutSteps.json');

  const expectation: HistoryNode[] = [
    {
      ...baseStepNode,
      attempt: 2,
      status: 'failed',
    },
    {
      ...baseRunEndNode,
      attempt: 2,
      status: 'failed',
    },
  ];

  expect(history).toEqual(expectation);
});

test('fails with preceding step', async () => {
  const history = await loadHistory('failsWithPrecedingStep.json');

  const expectation: HistoryNode[] = [
    {
      ...baseStepNode,
      status: 'completed',
      name: 'First step',
    },
    {
      ...baseStepNode,
      attempt: 2,
      status: 'failed',
    },
    {
      ...baseRunEndNode,
      attempt: 2,
      status: 'failed',
    },
  ];

  expect(history).toEqual(expectation);
});

test('no steps', async () => {
  const history = await loadHistory('noSteps.json');

  const expectation: HistoryNode[] = [
    {
      ...baseRunEndNode,
      status: 'completed',
    },
  ];

  expect(history).toEqual(expectation);
});

test('parallel steps', async () => {
  const history = await loadHistory('parallelSteps.json');

  const expectation: HistoryNode[] = [
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

  expect(history).toEqual(expectation);
});

test('sleeps', async () => {
  const history = await loadHistory('sleeps.json');

  const expectation: HistoryNode[] = [
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

  expect(history).toEqual(expectation);
});

test('succeeds with 2 steps', async () => {
  const history = await loadHistory('succeedsWith2Steps.json');

  const expectation: HistoryNode[] = [
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

  expect(history).toEqual(expectation);
});

test('times out waiting for events', async () => {
  const history = await loadHistory('timesOutWaitingForEvent.json');

  const expectation: HistoryNode[] = [
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

  expect(history).toEqual(expectation);
});

test('waits for event', async () => {
  const history = await loadHistory('waitsForEvent.json');

  const expectation: HistoryNode[] = [
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

  expect(history).toEqual(expectation);
});
