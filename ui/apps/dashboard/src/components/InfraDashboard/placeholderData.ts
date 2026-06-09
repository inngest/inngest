export type InfraPlanSku = 'IN-XS' | 'IN-S' | 'IN-M' | 'IN-L' | 'IN-XL';

export type InfraPlan = {
  sku: InfraPlanSku;
  eventStream: string;
  queueDepth: string;
  execConcurrency: string;
  priceMonthly: string;
};

export type InfraTierId = 'free' | 'shared' | 'dedicated';

export type InfraTier = {
  id: InfraTierId;
  name: string;
  description: string;
  availability: string;
  sla: string;
  dispatchP99: string;
  compliance?: string;
  notes?: string[];
};

export type InfraDashboardPlaceholders = {
  health: 'Healthy' | 'Degraded' | 'Down';
  region: string;
  version: string;
  defaultInfraTierId: InfraTierId;
  infraTiers: InfraTier[];
  eventRateLimit: {
    current: number;
    limit: number;
  };
  functionRateLimit: {
    current: number;
    limit: number;
  };
  lagMs: number;
  queueLatencyP50Ms: number;
  queueLatencyP99Ms: number;
  runsStateStoredBytes: number;
  runsStateStoredPercent: number;
  eventsStateStoredBytes: number;
  eventsStateStoredPercent: number;
  monthlyTracesReceived: number;
  monthlyScoresProcessed: number;
  defaultPlanSku: InfraPlanSku;
  infraPlans: InfraPlan[];
  deltas: {
    eventsReceivedPercent: number;
    functionsRanPercent: number;
    appsRegistered: number;
    functionsRegistered: number;
  };
};

// These values are present in the target design but are not exposed by the
// current dashboard GraphQL schema. Keep them isolated until API fields exist.
export const INFRA_DASHBOARD_PLACEHOLDERS: InfraDashboardPlaceholders = {
  health: 'Healthy',
  region: 'us-east-1',
  version: 'v2026.04.18',
  defaultInfraTierId: 'shared',
  infraTiers: [
    {
      id: 'free',
      name: 'Free pool',
      description: 'Included with IN-XS on the free tier',
      availability: 'Free',
      sla: '99.5%',
      dispatchP99: '< 2.5s',
    },
    {
      id: 'shared',
      name: 'Pro pool',
      description: 'Included with IN-S and up',
      availability: 'Included',
      sla: '99.9%',
      dispatchP99: '< 750ms',
    },
    {
      id: 'dedicated',
      name: 'Dedicated cluster',
      description: 'Your isolated event stream, queues, and executor infra',
      availability: 'from $2,899/mo',
      sla: '99.99%',
      dispatchP99: '< 500ms',
      compliance: 'SOC 2 / HIPAA',
      notes: [
        'Dedicated queue shards and execution infrastructure',
        'Custom retention and maintenance windows',
      ],
    },
  ],
  eventRateLimit: {
    current: 2,
    limit: 5,
  },
  functionRateLimit: {
    current: 2,
    limit: 5,
  },
  lagMs: 112,
  queueLatencyP50Ms: 142,
  queueLatencyP99Ms: 682,
  runsStateStoredBytes: 412 * 1024 ** 3,
  runsStateStoredPercent: 62,
  eventsStateStoredBytes: 1.4 * 1024 ** 4,
  eventsStateStoredPercent: 68,
  monthlyTracesReceived: 0,
  monthlyScoresProcessed: 0,
  defaultPlanSku: 'IN-S',
  infraPlans: [
    {
      sku: 'IN-XS',
      eventStream: '5 QPS',
      queueDepth: '100K',
      execConcurrency: '5',
      priceMonthly: '$0',
    },
    {
      sku: 'IN-S',
      eventStream: '50 QPS',
      queueDepth: '1M',
      execConcurrency: '100',
      priceMonthly: '$99',
    },
    {
      sku: 'IN-M',
      eventStream: '250 QPS',
      queueDepth: '5M',
      execConcurrency: '250',
      priceMonthly: '$249',
    },
    {
      sku: 'IN-L',
      eventStream: '500 QPS',
      queueDepth: '10M',
      execConcurrency: '500',
      priceMonthly: '$599',
    },
    {
      sku: 'IN-XL',
      eventStream: '1K QPS',
      queueDepth: '25M',
      execConcurrency: '1K',
      priceMonthly: '$1,199',
    },
  ],
  deltas: {
    eventsReceivedPercent: 6.2,
    functionsRanPercent: 6.2,
    appsRegistered: 1,
    functionsRegistered: 64,
  },
};
