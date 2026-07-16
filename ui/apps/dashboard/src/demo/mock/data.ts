/**
 * Deterministic domain data for the demo: environments, apps, functions, and
 * generators for runs / traces / time-series. Shapes follow the App API types
 * in src/gql/graphql.ts. Values are curated to look like a healthy, busy
 * account. Timestamps are anchored to "now" so charts read as recent, while
 * everything else is seeded and stable.
 */
import { Rng } from './prng';
import { seededULID, seededUUID } from './scalars';

export const now = () => Date.now();
const iso = (ms: number) => new Date(ms).toISOString();
const MIN = 60_000;
const HOUR = 60 * MIN;
const DAY = 24 * HOUR;

// --- Environments (workspaces) ---------------------------------------------

export type EnvDef = {
  slug: string;
  name: string;
  type: 'PRODUCTION' | 'BRANCH_PARENT' | 'BRANCH_CHILD' | 'TEST';
  test: boolean;
  parentSlug?: string;
};

export const ENVIRONMENTS: EnvDef[] = [
  { slug: 'production', name: 'Production', type: 'PRODUCTION', test: false },
  { slug: 'staging', name: 'Staging', type: 'BRANCH_PARENT', test: false },
  {
    slug: 'feat-new-checkout',
    name: 'feat/new-checkout',
    type: 'BRANCH_CHILD',
    test: false,
    parentSlug: 'staging',
  },
  { slug: 'my-test-env', name: 'My test env', type: 'TEST', test: true },
];

export function envBySlug(slug: string): EnvDef {
  return ENVIRONMENTS.find((e) => e.slug === slug) ?? ENVIRONMENTS[0];
}

export function workspaceFor(env: EnvDef, idHint?: string) {
  const id = seededUUID('env:' + env.slug);
  const createdAt = iso(now() - 120 * DAY);
  return {
    id: idHint ?? id,
    name: env.name,
    slug: env.slug,
    parentID: env.parentSlug ? seededUUID('env:' + env.parentSlug) : null,
    test: env.test,
    type: env.type,
    webhookSigningKey: `signkey-${env.slug}-${seededUUID(
      'whk:' + env.slug,
    ).slice(0, 12)}`,
    createdAt,
    lastDeployedAt: iso(
      now() - new Rng('deploy:' + env.slug).int(1, 48) * HOUR,
    ),
    isArchived: false,
    isAutoArchiveEnabled: env.type === 'BRANCH_CHILD',
    functionCount: APPS.reduce((n, a) => n + a.functions.length, 0),
  };
}

// --- Apps & functions -------------------------------------------------------

export type FnDef = {
  slug: string;
  name: string;
  trigger: { type: 'EVENT' | 'CRON'; value: string };
  paused?: boolean;
};

export type AppDef = {
  externalID: string;
  name: string;
  language: string;
  sdkVersion: string;
  framework: string;
  method: 'SERVE' | 'CONNECT' | 'API';
  functions: FnDef[];
};

export const APPS: AppDef[] = [
  {
    externalID: 'billing',
    name: 'billing',
    language: 'typescript',
    sdkVersion: '3.44.1',
    framework: 'nextjs',
    method: 'SERVE',
    functions: [
      {
        slug: 'process-invoice',
        name: 'Process invoice',
        trigger: { type: 'EVENT', value: 'billing/invoice.created' },
      },
      {
        slug: 'send-receipt',
        name: 'Send receipt',
        trigger: { type: 'EVENT', value: 'billing/payment.succeeded' },
      },
      {
        slug: 'retry-failed-charge',
        name: 'Retry failed charge',
        trigger: { type: 'EVENT', value: 'billing/payment.failed' },
      },
      {
        slug: 'nightly-reconciliation',
        name: 'Nightly reconciliation',
        trigger: { type: 'CRON', value: '0 2 * * *' },
      },
    ],
  },
  {
    externalID: 'email-engine',
    name: 'email-engine',
    language: 'typescript',
    sdkVersion: '3.44.1',
    framework: 'express',
    method: 'SERVE',
    functions: [
      {
        slug: 'send-welcome-email',
        name: 'Send welcome email',
        trigger: { type: 'EVENT', value: 'user/signed.up' },
      },
      {
        slug: 'weekly-digest',
        name: 'Weekly digest',
        trigger: { type: 'CRON', value: '0 14 * * 1' },
      },
      {
        slug: 'drip-campaign',
        name: 'Drip campaign',
        trigger: { type: 'EVENT', value: 'user/activated' },
      },
    ],
  },
  {
    externalID: 'user-workflows',
    name: 'user-workflows',
    language: 'python',
    sdkVersion: '0.4.15',
    framework: 'fastapi',
    method: 'CONNECT',
    functions: [
      {
        slug: 'onboard-user',
        name: 'Onboard user',
        trigger: { type: 'EVENT', value: 'user/signed.up' },
      },
      {
        slug: 'sync-crm',
        name: 'Sync to CRM',
        trigger: { type: 'EVENT', value: 'user/updated' },
      },
      {
        slug: 'offboard-user',
        name: 'Offboard user',
        trigger: { type: 'EVENT', value: 'user/deleted' },
      },
    ],
  },
  {
    externalID: 'data-pipeline',
    name: 'data-pipeline',
    language: 'go',
    sdkVersion: '0.11.0',
    framework: 'connect',
    method: 'CONNECT',
    functions: [
      {
        slug: 'ingest-events',
        name: 'Ingest events',
        trigger: { type: 'EVENT', value: 'data/batch.received' },
      },
      {
        slug: 'aggregate-metrics',
        name: 'Aggregate metrics',
        trigger: { type: 'CRON', value: '*/15 * * * *' },
      },
      {
        slug: 'export-warehouse',
        name: 'Export to warehouse',
        trigger: { type: 'CRON', value: '0 * * * *' },
      },
      {
        slug: 'cleanup-stale',
        name: 'Cleanup stale rows',
        trigger: { type: 'CRON', value: '0 3 * * *' },
        paused: true,
      },
    ],
  },
];

export function allFunctions(): { app: AppDef; fn: FnDef }[] {
  return APPS.flatMap((app) => app.functions.map((fn) => ({ app, fn })));
}

export function deployFor(app: AppDef, envSlug: string) {
  const rng = new Rng(`sync:${app.externalID}:${envSlug}`);
  return {
    id: seededUUID(`sync:${app.externalID}:${envSlug}`),
    appVersion: `2024.11.${rng.int(1, 40)}`,
    commitAuthor: rng.pick(['jane', 'sam', 'ada', 'leo']),
    commitHash: seededUUID('hash:' + app.externalID)
      .replace(/-/g, '')
      .slice(0, 40),
    commitMessage: rng.pick([
      'fix reton backoff',
      'add new step',
      'bump sdk',
      'improve logging',
    ]),
    commitRef: 'main',
    error: null,
    framework: app.framework,
    lastSyncedAt: iso(now() - rng.int(1, 72) * HOUR),
    platform: rng.pick(['vercel', 'aws-lambda', 'render', 'railway']),
    repoURL: `https://github.com/acme/${app.externalID}`,
    sdkLanguage: app.language,
    sdkVersion: app.sdkVersion,
    status: 'success',
    url: `https://${app.externalID}.acme.com/api/inngest`,
    vercelDeploymentID: null,
    vercelDeploymentURL: null,
    vercelProjectID: null,
    vercelProjectURL: null,
  };
}

// --- Runs -------------------------------------------------------------------

const RUN_STATUS_WEIGHTS = [
  ['COMPLETED', 86],
  ['RUNNING', 6],
  ['FAILED', 4],
  ['QUEUED', 3],
  ['CANCELLED', 1],
] as const;

export type RunSeed = {
  id: string;
  status: string;
  app: AppDef;
  fn: FnDef;
  queuedAt: number;
  startedAt: number | null;
  endedAt: number | null;
};

// Fixed epoch for run-id ULIDs. The id must be identical across requests so
// the run list and a later run-detail request agree; anchoring it to a
// constant (not `now()`) keeps ids stable while display timestamps below stay
// anchored to `now()` for recency.
const RUN_ID_EPOCH = 1_700_000_000_000;

/** Deterministic list of runs, newest first, for a given env. */
export function runsFor(envSlug: string, count = 40): RunSeed[] {
  const fns = allFunctions();
  const out: RunSeed[] = [];
  for (let i = 0; i < count; i++) {
    const rng = new Rng(`run:${envSlug}:${i}`);
    const { app, fn } = rng.pick(fns);
    const status = fn.paused ? 'CANCELLED' : rng.weighted(RUN_STATUS_WEIGHTS);
    const queuedAt = now() - i * rng.int(20_000, 90_000);
    const started = status === 'QUEUED' ? null : queuedAt + rng.int(50, 1500);
    const durationMs = rng.int(180, 45_000);
    const ended =
      status === 'RUNNING' || status === 'QUEUED'
        ? null
        : (started ?? queuedAt) + durationMs;
    out.push({
      // Stable id (fixed epoch, count-independent so list & detail agree).
      // Newer runs (smaller i) get a larger timestamp so ULID lexical order
      // matches the newest-first display order.
      id: seededULID(`run:${envSlug}:${i}`, RUN_ID_EPOCH + (10_000 - i) * 1000),
      status,
      app,
      fn,
      queuedAt,
      startedAt: started,
      endedAt: ended,
    });
  }
  return out;
}

/** Find a run seed by id within an env (searches a wide window). */
export function findRunSeed(
  envSlug: string,
  runID: string,
): RunSeed | undefined {
  return runsFor(envSlug, 120).find((s) => s.id === runID);
}

/**
 * Deterministic 24h per-function volume, split into completed / cancelled /
 * failed counts with a healthy (~2-5%) failure rate. Drives the functions-list
 * failure-rate + volume columns and the function-detail usage charts.
 */
export function functionDailyUsage(app: AppDef, fn: FnDef) {
  const rng = new Rng(`usage:${app.externalID}:${fn.slug}`);
  const points = 24;
  const start = now() - (points - 1) * HOUR;
  // Per-hour started counts as a smooth curve; paused fns are quiet.
  const base = fn.paused ? 0 : rng.int(40, 220);
  const amp = Math.round(base * 0.35);
  const started = Array.from({ length: points }, (_, i) => {
    const trend = base * (1 + (i / points) * 0.15);
    const daily = amp * Math.sin((i / points) * Math.PI * 4);
    const noise = rng.float(-amp * 0.15, amp * 0.15);
    return {
      slot: iso(start + i * HOUR),
      count: Math.max(0, Math.round(trend + daily + noise)),
    };
  });
  const totalStarted = started.reduce((n, s) => n + s.count, 0);
  const failureRate = fn.paused ? 0 : rng.float(0.01, 0.05);
  const cancelRate = fn.paused ? 0 : rng.float(0.002, 0.01);
  const failed = started.map((s) => ({
    slot: s.slot,
    count: Math.round(s.count * failureRate),
  }));
  const cancelled = started.map((s) => ({
    slot: s.slot,
    count: Math.round(s.count * cancelRate),
  }));
  const totalFailed = failed.reduce((n, s) => n + s.count, 0);
  const totalCancelled = cancelled.reduce((n, s) => n + s.count, 0);
  const totalCompleted = Math.max(
    0,
    totalStarted - totalFailed - totalCancelled,
  );
  const completed = started.map((s, i) => ({
    slot: s.slot,
    count: Math.max(0, s.count - failed[i].count - cancelled[i].count),
  }));
  return {
    from: iso(start),
    to: iso(now()),
    started,
    completed,
    cancelled,
    failed,
    totals: {
      started: totalStarted,
      completed: totalCompleted,
      cancelled: totalCancelled,
      failed: totalFailed,
    },
  };
}

export const runIso = iso;
export { HOUR as HOUR_MS };
