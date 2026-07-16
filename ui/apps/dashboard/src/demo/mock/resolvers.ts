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
  eventEnvelope,
  decodeOutputID,
  encodeOutputID,
  runError,
  runOutput,
  stepOutput,
  stepsFor,
} from './content';
import {
  allFunctions,
  APPS,
  type AppDef,
  deployFor,
  ENVIRONMENTS,
  envBySlug,
  findRunSeed,
  type FnDef,
  functionDailyUsage,
  now,
  runIso,
  runsFor,
  type RunSeed,
  workspaceFor,
} from './data';
import { Rng } from './prng';
import { seededULID, seededUUID } from './scalars';

const asBytes = (obj: unknown) => JSON.stringify(obj);

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
  const stepNames = stepsFor(seed);
  const failed = seed.status === 'FAILED';
  const running = seed.status === 'RUNNING';
  const q = seed.queuedAt;
  const start = seed.startedAt ?? q + 200;
  const end = seed.endedAt ?? start + 8000;
  const span = seededUUID('span:' + seed.id);
  const total = end - start;
  const n = stepNames.length;
  const children = stepNames.map((stepName, i) => {
    const cs = start + (total / n) * i;
    const ce = start + (total / n) * (i + 1);
    const isLast = i === n - 1;
    // A failed run fails on its last step; a running run's last step is live.
    const status =
      failed && isLast ? 'FAILED' : running && isLast ? 'RUNNING' : 'COMPLETED';
    const isWait = stepName.startsWith('wait') || stepName === 'sleep';
    const stepOp = isWait
      ? 'SLEEP'
      : stepName === 'sync-crm' || stepName === 'provision'
      ? 'INVOKE'
      : 'RUN';
    return {
      spanID: seededUUID(`span:${seed.id}:${i}`),
      name: stepName,
      status,
      isRoot: false,
      isUserland: false,
      stepOp,
      stepType: 'step.run',
      stepID: stepName,
      attempts: failed && isLast ? rng.int(2, 4) : 1,
      queuedAt: runIso(cs - 30),
      scheduledAt: runIso(cs - 15),
      startedAt: runIso(cs),
      endedAt: running && isLast ? null : runIso(ce),
      duration: running && isLast ? null : Math.round(ce - cs),
      childrenSpans: [],
      metadata: [],
      traceID: seededUUID('trace:' + seed.id),
      runID: seed.id,
      // Decodable so runTraceSpanOutputByID can return matching step output.
      outputID: encodeOutputID(seed.id, `step:${i}:${stepName}`),
      parentSpanID: span,
    };
  });
  return {
    spanID: span,
    name: seed.fn.name,
    status: failed ? 'FAILED' : running ? 'RUNNING' : 'COMPLETED',
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
    outputID: encodeOutputID(seed.id, 'root'),
    parentSpanID: null,
  };
}

// --- scoped metrics ---------------------------------------------------------

type ScopedFilter = {
  name?: string;
  scope?: string;
  groupBy?: string;
  functionIDs?: string[];
  appIDs?: string[];
  from?: string;
  until?: string;
};

/** A metrics series bounded to an explicit [from,to] window. */
function scopedSeries(
  seed: string,
  from: number,
  to: number,
  base: number,
  amp: number,
) {
  const points = 24;
  const step = Math.max(1, (to - from) / (points - 1));
  const rng = new Rng('scoped:' + seed);
  return Array.from({ length: points }, (_, i) => {
    const trend = base * (1 + (i / points) * 0.2);
    const daily = amp * Math.sin((i / points) * Math.PI * 4);
    const noise = rng.float(-amp * 0.15, amp * 0.15);
    return {
      bucket: runIso(from + i * step),
      value: Math.max(0, Math.round(trend + daily + noise)),
    };
  });
}

// Status breakdown shares (healthy account).
const STATUS_SHARES: [string, number][] = [
  ['Completed', 0.94],
  ['Failed', 0.04],
  ['Cancelled', 0.02],
];

function buildScopedMetrics(envSlug: string, filter?: ScopedFilter) {
  const name = filter?.name ?? 'function_run_scheduled_total';
  const scope = filter?.scope ?? 'ENV';
  const groupBy = filter?.groupBy;
  const from = filter?.from
    ? new Date(filter.from).getTime()
    : now() - 24 * HOUR;
  const to = filter?.until ? new Date(filter.until).getTime() : now();
  const base = name.includes('step')
    ? 1800
    : name.includes('started')
    ? 880
    : name.includes('ended')
    ? 850
    : 900;
  const envelope = (metrics: unknown[]) => ({
    from: runIso(from),
    to: runIso(to),
    granularity: '1h',
    scope,
    metrics,
  });

  if (groupBy === 'status' && scope === 'FN') {
    // Per function × status (drives Failed-functions breakdown).
    const metrics = allFunctions().flatMap(({ app, fn }) => {
      const fid = seededUUID(`fn:${app.externalID}:${fn.slug}:${envSlug}`);
      const u = functionDailyUsage(app, fn);
      const perStatus: [string, number][] = [
        ['Completed', u.totals.completed],
        ['Failed', u.totals.failed],
        ['Cancelled', u.totals.cancelled],
      ];
      return perStatus.map(([status, tot]) => ({
        id: fid,
        tagName: 'status',
        tagValue: status,
        data: scopedSeries(
          `${fn.slug}:${status}`,
          from,
          to,
          Math.max(1, tot / 24),
          Math.max(1, tot / 80),
        ),
      }));
    });
    return envelope(metrics);
  }

  if (groupBy === 'status') {
    return envelope(
      STATUS_SHARES.map(([status, frac]) => ({
        id: seededUUID(`status:${status}:${envSlug}`),
        tagName: 'status',
        tagValue: status,
        data: scopedSeries(status, from, to, base * frac, base * frac * 0.3),
      })),
    );
  }

  // Ungrouped: one series (optionally per function when scope=FN).
  if (scope === 'FN') {
    return envelope(
      allFunctions().map(({ app, fn }) => ({
        id: seededUUID(`fn:${app.externalID}:${fn.slug}:${envSlug}`),
        tagName: null,
        tagValue: null,
        data: scopedSeries(
          fn.slug + name,
          from,
          to,
          functionDailyUsage(app, fn).totals.started / 24,
          base * 0.1,
        ),
      })),
    );
  }
  return envelope([
    {
      id: seededUUID(`scoped:${name}:${envSlug}`),
      tagName: null,
      tagValue: null,
      data: scopedSeries(name, from, to, base, base * 0.25),
    },
  ]);
}

// --- workflow usage ---------------------------------------------------------

function buildUsage(app: AppDef, fn: FnDef, event: string | null) {
  const u = functionDailyUsage(app, fn);
  const pick =
    event === 'completed'
      ? u.completed
      : event === 'cancelled'
      ? u.cancelled
      : event === 'errored' || event === 'failed'
      ? u.failed
      : u.started;
  const total =
    event === 'completed'
      ? u.totals.completed
      : event === 'cancelled'
      ? u.totals.cancelled
      : event === 'errored' || event === 'failed'
      ? u.totals.failed
      : u.totals.started;
  return {
    asOf: runIso(now()),
    period: 'hour',
    total,
    range: { start: u.from, end: u.to },
    data: pick,
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
      // Healthy entitlements: seats well under limit (clears the seat-overage
      // widget), generous run/step limits with modest usage. Unspecified
      // nested fields fall back to auto-mocks.
      entitlements: () => ({
        userCount: { usage: 6, limit: 25 },
        runCount: { usage: 184_000, limit: 5_000_000, overageAllowed: true },
        stepCount: { usage: 921_500, limit: 25_000_000, overageAllowed: true },
        concurrency: { limit: 100 },
        history: { limit: 7 },
        eventSize: { limit: 3_145_728 },
        metricsExportFreshness: { limit: 900 },
        metricsExportGranularity: { limit: 3600 },
        connectWorkerConnections: { limit: 100 },
        hipaa: { enabled: false },
        metricsExport: { enabled: true },
        slackChannel: { enabled: true },
      }),
      entitlementUsage: () => ({
        accountConcurrencyLimitHits: 0,
        runCount: {
          current: 184_000,
          limit: 5_000_000,
          overageAllowed: true,
        },
        stepCount: {
          current: 921_500,
          limit: 25_000_000,
          overageAllowed: true,
        },
      }),
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
      workflow: (ws: { slug?: string }, { id }: { id: string }) => {
        const envSlug = wsSlug(ws);
        const found = allFunctions().find(
          ({ app, fn }) =>
            seededUUID(`fn:${app.externalID}:${fn.slug}:${envSlug}`) === id,
        );
        return found ? buildWorkflow(found.app, found.fn, envSlug) : null;
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
        const seed = findRunSeed(envSlug, runID) ?? runsFor(envSlug, 1)[0];
        return buildRun({ ...seed, id: runID }, envSlug);
      },
      runTrigger: (ws: { slug?: string }, { runID }: { runID: string }) => {
        const envSlug = wsSlug(ws);
        const seed = findRunSeed(envSlug, runID) ?? runsFor(envSlug, 1)[0];
        const isCron = seed.fn.trigger.type === 'CRON';
        return {
          IDs: [seededULID('trig:' + runID, seed.queuedAt)],
          payloads: [asBytes(eventEnvelope(seed))],
          timestamp: runIso(seed.queuedAt),
          eventName: isCron ? null : seed.fn.trigger.value,
          isBatch: false,
          batchID: null,
          cron: isCron ? seed.fn.trigger.value : null,
        };
      },
      runTraceSpanOutputByID: (
        ws: { slug?: string },
        { outputID }: { outputID: string },
      ) => {
        const envSlug = wsSlug(ws);
        const { runID, kind } = decodeOutputID(outputID);
        const seed = findRunSeed(envSlug, runID);
        if (!seed) return { data: null, input: null, error: null };

        const stepNames = stepsFor(seed);
        const isRoot = kind === 'root';
        const stepIdx = kind.startsWith('step:')
          ? Number(kind.split(':')[1])
          : -1;
        const isLastStep = stepIdx === stepNames.length - 1;
        const failedHere = seed.status === 'FAILED' && (isRoot || isLastStep);

        if (failedHere) {
          return {
            data: null,
            input: asBytes(eventEnvelope(seed).data),
            error: runError(seed),
          };
        }
        if (isRoot) {
          return {
            data: asBytes(runOutput(seed)),
            input: asBytes(eventEnvelope(seed).data),
            error: null,
          };
        }
        const stepName = stepNames[stepIdx] ?? 'step';
        return {
          data: asBytes(stepOutput(seed, stepName)),
          input: null,
          error: null,
        };
      },
      scopedMetrics: (
        ws: { slug?: string },
        { filter }: { filter?: ScopedFilter },
      ) => buildScopedMetrics(wsSlug(ws), filter),
      scopedFunctionStatus: (ws: { slug?: string }) => {
        // Aggregate healthy totals across all functions in the env.
        const totals = allFunctions().reduce(
          (acc, { app, fn }) => {
            const u = functionDailyUsage(app, fn);
            acc.completed += u.totals.completed;
            acc.failed += u.totals.failed;
            acc.cancelled += u.totals.cancelled;
            return acc;
          },
          { completed: 0, failed: 0, cancelled: 0 },
        );
        const rng = new Rng('status:' + wsSlug(ws));
        return {
          from: runIso(now() - 24 * HOUR),
          to: runIso(now()),
          completed: totals.completed,
          failed: totals.failed,
          cancelled: totals.cancelled,
          running: rng.int(3, 24),
          queued: rng.int(1, 12),
          skipped: rng.int(0, 4),
        };
      },
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
      usage: (
        wf: { _app: AppDef; _fn: FnDef },
        { event }: { event?: string | null },
      ) => buildUsage(wf._app, wf._fn, event ?? null),
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
