import { FunctionRunStatus, FunctionTriggerTypes } from '../src/store/generated';

export const triggerStream = [
  {
    id: 'id1',
    name: 'payment.intent.created',
    type: FunctionTriggerTypes.Event,
    startedAt: '2023-07-17T00:08:42.725+01:00',
    source: {
      type: 'webhook',
      name: 'Stripe',
    },
    test: true,
    functionRuns: [
      {
        id: 'function1',
        name: 'Email: Make Payment',
        status: FunctionRunStatus.Completed,
      },
      {
        id: 'function2',
        name: 'Discord: Send Notification',
        status: FunctionRunStatus.Running,
      },
    ],
  },
  {
    id: 'id2',
    name: 'password.created',
    type: FunctionTriggerTypes.Event,
    startedAt: '2023-07-17T00:07:42.725+01:00',
    source: {
      type: 'app',
      name: 'Dashboard',
    },
    functionRuns: [],
  },
  {
    id: 'id3',
    name: 'TZ=Europe/Paris 0 12 * * 5',
    type: FunctionTriggerTypes.Cron,
    startedAt: '2023-07-17T00:07:42.725+01:00',
    source: {
      type: 'manual',
    },
    functionRuns: [
      {
        id: 'function3',
        name: 'Email: Changed Timezone',
        status: FunctionRunStatus.Failed,
      },
    ],
  },
];
