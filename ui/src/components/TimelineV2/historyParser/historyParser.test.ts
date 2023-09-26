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

test('cancels', async () => {
  const history = await loadHistory('cancels.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 0,
      groupID: expect.any(String),
      status: 'cancelled',
      scheduledAt: expect.any(Date),
      startedAt: expect.any(Date),
      scope: 'step',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
      name: '1m',
      sleepConfig: {
        until: expect.any(Date),
      },
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'cancelled',
      endedAt: expect.any(Date),
      scope: 'function',
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});

test('fails without steps', async () => {
  const history = await loadHistory('failsWithoutSteps.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 2,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'failed',
      startedAt: expect.any(Date),
      scope: 'step',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
    {
      attempt: 2,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'failed',
      startedAt: expect.any(Date),
      scope: 'function',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});

test('fails with preceding step', async () => {
  const history = await loadHistory('failsWithPrecedingStep.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'step',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
      name: 'First step',
    },
    {
      attempt: 2,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'failed',
      scope: 'step',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
    {
      attempt: 2,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'failed',
      scope: 'function',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});

test('no steps', async () => {
  const history = await loadHistory('noSteps.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'function',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});

test('parallel steps', async () => {
  const history = await loadHistory('parallelSteps.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 0,
      groupID: expect.any(String),
      status: 'completed',
      scheduledAt: expect.any(Date),
      startedAt: expect.any(Date),
      scope: 'step',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
      name: 'a',
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      status: 'completed',
      scheduledAt: expect.any(Date),
      scope: 'step',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      status: 'completed',
      name: 'b2',
      scheduledAt: expect.any(Date),
      scope: 'step',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      status: 'completed',
      name: 'b1',
      scheduledAt: expect.any(Date),
      scope: 'step',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      status: 'completed',
      scheduledAt: expect.any(Date),
      scope: 'step',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      status: 'completed',
      scheduledAt: expect.any(Date),
      scope: 'function',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});

test('sleeps', async () => {
  const history = await loadHistory('sleeps.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 0,
      endedAt: expect.any(Date),
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'step',
      url: 'http://localhost:3939/api/inngest',
      name: '10s',
      sleepConfig: {
        until: expect.any(Date),
      },
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'function',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});

test('succeeds with 2 steps', async () => {
  const history = await loadHistory('succeedsWith2Steps.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'step',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
      name: 'First step',
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      name: 'Second step',
      scope: 'step',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      scope: 'function',
      startedAt: expect.any(Date),
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});

test('times out waiting for events', async () => {
  const history = await loadHistory('timesOutWaitingForEvent.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'step',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
      waitForEventResult: {
        eventID: undefined,
        timeout: true,
      },
      waitForEventConfig: {
        eventName: 'foo',
        expression: undefined,
        timeout: expect.any(Date),
      },
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'function',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});

test('waits for event', async () => {
  const history = await loadHistory('waitsForEvent.json');

  const expectation: HistoryNode[] = [
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'step',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
      waitForEventResult: {
        eventID: '01HAW74MMPVBF8RJSTGQHTVJFR',
        timeout: false,
      },
      waitForEventConfig: {
        eventName: 'bar',
        expression: undefined,
        timeout: expect.any(Date),
      },
    },
    {
      attempt: 0,
      groupID: expect.any(String),
      scheduledAt: expect.any(Date),
      status: 'completed',
      startedAt: expect.any(Date),
      scope: 'function',
      url: 'http://localhost:3939/api/inngest',
      endedAt: expect.any(Date),
    },
  ];

  expect(history).toEqual(
    expectation.map((exp) => {
      return expect.objectContaining(exp);
    }),
  );
});
