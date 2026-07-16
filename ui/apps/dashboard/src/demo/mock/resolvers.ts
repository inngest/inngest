/**
 * Resolvers layered over the auto-mocked schema (see index.ts). These cover the
 * bootstrap path (account, environments) and the hero screens (Apps &
 * Functions, Runs & Traces, Metrics & Insights). Anything not resolved here
 * falls back to the deterministic auto-mocks, so the whole schema still
 * responds. Objects carry `_`-prefixed context fields (envSlug, ids) so child
 * type resolvers can rebuild their data; GraphQL ignores those extra fields.
 */
import { DEMO_ACCOUNT_ID } from '../clerk/identity';
import {
  allFunctions,
  APPS,
  type AppDef,
  deployFor,
  ENVIRONMENTS,
  envBySlug,
  type FnDef,
  now,
  runIso,
  runsFor,
  type RunSeed,
  workspaceFor,
} from './data';
import { Rng } from './prng';
import { seededULID, seededUUID } from './scalars';

const HOUR = 3_600_000;
const DAY = 24 * HOUR;

// --- builders ---------------------------------------------------------------

function buildApp(app: AppDef, envSlug: string) {
  return {
    _envSlug: envSlug,
    _app: app,
    id: seededUUID(`app:${app.externalID}:${envSlug}`),
    externalID: app.externalID,
    name: app.name,
    method: app.method,
    appVersion: `2024.11`,
    functionCount: app.functions.length,
    isArchived: false,
    isParentArchived: false,
    archivedAt: null,
    createdAt: runIso(now() - 90 * DAY),
  };
}

function buildWorkflow(app: AppDef, fn: FnDef, envSlug: string) {
  return {
    _envSlug: envSlug,
    _app: app,
    _fn: fn,
    id: seededUUID(`fn:${app.externalID}:${fn.slug}:${envSlug}`),
    name: fn.name,
    slug: fn.slug,
    isPaused: Boolean(fn.paused),
    isArchived: false,
    isParentArchived: false,
    archivedAt: null,
    keyQueuesEnabled: false,
    cancellationRunCount: 0,
    url: `https://${app.externalID}.acme.com/api/inngest?fnId=${fn.slug}`,
  };
}

function buildRun(seed: RunSeed, envSlug: string) {
  return {
    _envSlug: envSlug,
    _seed: seed,
    id: seed.id,
    status: seed.status,
    eventName: seed.fn.trigger.type === 'EVENT' ? seed.fn.trigger.value : null,
    cronSchedule:
      seed.fn.trigger.type === 'CRON' ? seed.fn.trigger.value : null,
    isBatch: false,
    isDeferred: false,
    hasAI: seed.app.externalID === 'email-engine',
    queuedAt: runIso(seed.queuedAt),
    startedAt: seed.startedAt ? runIso(seed.startedAt) : null,
    endedAt: seed.endedAt ? runIso(seed.endedAt) : null,
    traceID: seededUUID('trace:' + seed.id),
    triggerIDs: [seededULID('trig:' + seed.id, seed.queuedAt)],
    accountID: DEMO_ACCOUNT_ID,
    appID: seededUUID(`app:${seed.app.externalID}:${envSlug}`),
    functionID: seededUUID(
      `fn:${seed.app.externalID}:${seed.fn.slug}:${envSlug}`,
    ),
    workspaceID: seededUUID('env:' + envSlug),
    deferredFrom: [],
    defers: [],
    siblingDefers: [],
    sourceID: null,
    batchCreatedAt: null,
    output: null,
  };
}

// --- time series ------------------------------------------------------------

/** Smooth healthy curve: upward trend + daily seasonality + gentle noise. */
function series(seed: string, points: number, base: number, amp: number) {
  const rng = new Rng('series:' + seed);
  const start = now() - (points - 1) * HOUR;
  return Array.from({ length: points }, (_, i) => {
    const trend = base * (1 + (i / points) * 0.4);
    const daily = amp * Math.sin((i / points) * Math.PI * 4);
    const noise = rng.float(-amp * 0.15, amp * 0.15);
    return {
      time: runIso(start + i * HOUR),
      bucket: runIso(start + i * HOUR),
      value: Math.max(0, Math.round(trend + daily + noise)),
    };
  });
}

function metricsResponse(seed: string, base: number, amp: number) {
  const points = 24;
  const start = now() - (points - 1) * HOUR;
  return {
    from: runIso(start),
    to: runIso(now()),
    granularity: '1h',
    data: series(seed, points, base, amp).map((p) => ({
      bucket: p.bucket,
      value: p.value,
    })),
  };
}

// --- trace ------------------------------------------------------------------

function buildTrace(seed: RunSeed, envSlug: string) {
  const rng = new Rng('span:' + seed.id);
  const stepNames = ['load-context', 'call-api', 'persist-result', 'notify'];
  const stepCount = rng.int(2, 4);
  const q = seed.queuedAt;
  const start = seed.startedAt ?? q + 200;
  const end = seed.endedAt ?? start + 8000;
  const span = seededUUID('span:' + seed.id);
  const total = end - start;
  const children = Array.from({ length: stepCount }, (_, i) => {
    const cs = start + (total / stepCount) * i;
    const ce = start + (total / stepCount) * (i + 1);
    return {
      spanID: seededUUID(`span:${seed.id}:${i}`),
      name: stepNames[i % stepNames.length],
      status: 'COMPLETED',
      isRoot: false,
      isUserland: false,
      stepOp: rng.pick(['RUN', 'RUN', 'INVOKE', 'SLEEP', 'WAIT_FOR_EVENT']),
      stepType: 'step.run',
      attempts: 1,
      queuedAt: runIso(cs - 30),
      scheduledAt: runIso(cs - 15),
      startedAt: runIso(cs),
      endedAt: runIso(ce),
      duration: Math.round(ce - cs),
      childrenSpans: [],
      metadata: [],
      traceID: seededUUID('trace:' + seed.id),
      runID: seed.id,
    };
  });
  return {
    spanID: span,
    name: seed.fn.name,
    status:
      seed.status === 'FAILED'
        ? 'FAILED'
        : seed.status === 'RUNNING'
        ? 'RUNNING'
        : 'COMPLETED',
    isRoot: true,
    isUserland: false,
    stepType: 'run',
    attempts: 1,
    queuedAt: runIso(q),
    scheduledAt: runIso(q + 20),
    startedAt: runIso(start),
    endedAt: seed.endedAt ? runIso(end) : null,
    duration: seed.endedAt ? Math.round(end - start) : null,
    childrenSpans: children,
    metadata: [],
    traceID: seededUUID('trace:' + seed.id),
    runID: seed.id,
    accountID: DEMO_ACCOUNT_ID,
    appID: seededUUID(`app:${seed.app.externalID}:${envSlug}`),
    functionID: seededUUID(
      `fn:${seed.app.externalID}:${seed.fn.slug}:${envSlug}`,
    ),
    workspaceID: seededUUID('env:' + envSlug),
    outputID: seededUUID('out:' + seed.id),
    parentSpanID: null,
  };
}

// --- resolvers --------------------------------------------------------------

const pageInfo = {
  hasNextPage: false,
  hasPreviousPage: false,
  startCursor: null,
  endCursor: null,
};

export function buildResolvers() {
  const account = {
    id: DEMO_ACCOUNT_ID,
    name: 'Acme, Inc.',
    marketplace: null,
    status: 'active',
    billingEmail: 'billing@acme.com',
    createdAt: runIso(now() - 200 * DAY),
    updatedAt: runIso(now()),
  };

  const workspaceById = (id: string) => {
    const env =
      ENVIRONMENTS.find((e) => seededUUID('env:' + e.slug) === id) ??
      ENVIRONMENTS[0];
    return workspaceFor(env, id);
  };

  return {
    Query: {
      account: () => account,
      workspaces: () => ENVIRONMENTS.map((e) => workspaceFor(e)),
      defaultEnv: () => workspaceFor(ENVIRONMENTS[0]),
      envBySlug: (_: unknown, { slug }: { slug: string }) =>
        workspaceFor(envBySlug(slug)),
      workspace: (_: unknown, { id }: { id: string }) => workspaceById(id),
      runCountTimeSeries: () => [
        { name: 'Completed', data: series('runs-completed', 48, 800, 220) },
        { name: 'Failed', data: series('runs-failed', 48, 30, 18) },
      ],
      executionTimeSeries: () => [
        { name: 'Executions', data: series('executions', 48, 2400, 500) },
      ],
      billableStepTimeSeries: () => [
        { name: 'Steps', data: series('steps', 48, 5200, 900) },
      ],
      metrics: () => metricsResponse('metrics', 800, 220),
    },

    Account: {
      // No auto-mocked "Hello World" banners in the demo.
      activeBanners: () => [],
    },

    Workspace: {
      apps: (ws: { id: string; slug?: string }) =>
        APPS.map((a) => buildApp(a, wsSlug(ws))),
      workflows: (ws: { id: string; slug?: string }) => ({
        data: allFunctions().map(({ app, fn }) =>
          buildWorkflow(app, fn, wsSlug(ws)),
        ),
        page: {
          page: 1,
          perPage: 100,
          totalItems: allFunctions().length,
          totalPages: 1,
        },
      }),
      workflowBySlug: (ws: { slug?: string }, { slug }: { slug: string }) => {
        const found = allFunctions().find((f) => f.fn.slug === slug);
        return found ? buildWorkflow(found.app, found.fn, wsSlug(ws)) : null;
      },
      runs: (ws: { slug?: string }) => {
        const envSlug = wsSlug(ws);
        const seeds = runsFor(envSlug);
        return {
          totalCount: seeds.length,
          pageInfo,
          edges: seeds.map((s) => ({
            cursor: s.id,
            node: buildRun(s, envSlug),
          })),
        };
      },
      run: (ws: { slug?: string }, { runID }: { runID: string }) => {
        const envSlug = wsSlug(ws);
        const seeds = runsFor(envSlug, 60);
        const seed = seeds.find((s) => s.id === runID) ?? seeds[0];
        return buildRun({ ...seed, id: runID }, envSlug);
      },
      scopedMetrics: () => ({
        metrics: [
          {
            tagName: null,
            tagValue: null,
            data: metricsResponse('scoped', 800, 220).data,
          },
        ],
      }),
      functionCount: () => allFunctions().length,
    },

    App: {
      functions: (app: { _app: AppDef; _envSlug: string }) =>
        app._app.functions.map((fn) =>
          buildWorkflow(app._app, fn, app._envSlug),
        ),
      latestSync: (app: { _app: AppDef; _envSlug: string }) =>
        deployFor(app._app, app._envSlug),
      syncs: (app: { _app: AppDef; _envSlug: string }) => [
        deployFor(app._app, app._envSlug),
      ],
    },

    Workflow: {
      app: (wf: { _app: AppDef; _envSlug: string }) =>
        buildApp(wf._app, wf._envSlug),
      triggers: (wf: { _fn: FnDef }) => [
        { type: wf._fn.trigger.type, value: wf._fn.trigger.value },
      ],
      metrics: (wf: { _fn: FnDef }) =>
        metricsResponse('fn:' + wf._fn.slug, 120, 40),
    },

    FunctionRunV2: {
      app: (run: { _seed: RunSeed; _envSlug: string }) =>
        buildApp(run._seed.app, run._envSlug),
      function: (run: { _seed: RunSeed; _envSlug: string }) =>
        buildWorkflow(run._seed.app, run._seed.fn, run._envSlug),
      trace: (run: { _seed: RunSeed; _envSlug: string }) =>
        buildTrace(run._seed, run._envSlug),
    },
  };
}

/** Recover the env slug from a resolved workspace object. */
function wsSlug(ws: { slug?: string; id?: string }): string {
  if (ws.slug) return ws.slug;
  const env = ENVIRONMENTS.find((e) => seededUUID('env:' + e.slug) === ws.id);
  return env?.slug ?? 'production';
}
