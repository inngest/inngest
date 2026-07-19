/**
 * Realistic, domain-specific event payloads and step outputs for the demo run
 * views. Keyed off the app + function so the trigger panel, step I/O, and error
 * details read like a real production account rather than auto-mock filler.
 */
import type { RunSeed } from './data';
import { Rng } from './prng';
import { seededULID } from './scalars';

type Json = Record<string, unknown>;

// --- span output-id codec ---------------------------------------------------
// A run-trace span's `outputID` is later passed back to
// runTraceSpanOutputByID. We encode which run + step it refers to so the
// output resolver can return matching data/error. ULIDs contain no '~'.

export function encodeOutputID(runID: string, kind: string): string {
  return `${runID}~${kind}`;
}

export function decodeOutputID(token: string): { runID: string; kind: string } {
  const idx = token.indexOf('~');
  if (idx === -1) return { runID: token, kind: 'root' };
  return { runID: token.slice(0, idx), kind: token.slice(idx + 1) };
}

/** Function-appropriate step names for a run's trace. */
export function stepsFor(seed: RunSeed): string[] {
  const byFn: Record<string, string[]> = {
    'process-invoice': [
      'load-invoice',
      'charge-card',
      'persist-result',
      'notify',
    ],
    'send-receipt': ['load-invoice', 'render-pdf', 'send-email'],
    'retry-failed-charge': ['load-invoice', 'charge-card'],
    'nightly-reconciliation': ['load-context', 'reconcile', 'persist-result'],
    'send-welcome-email': ['load-user', 'render-email', 'send-email'],
    'weekly-digest': [
      'load-context',
      'aggregate',
      'render-email',
      'send-email',
    ],
    'drip-campaign': ['load-user', 'send-email'],
    'onboard-user': ['load-user', 'provision', 'sync-crm', 'notify'],
    'sync-crm': ['load-user', 'sync-crm'],
    'offboard-user': ['load-user', 'revoke-access', 'notify'],
    'ingest-events': ['load-context', 'validate', 'persist-result'],
    'aggregate-metrics': ['load-context', 'aggregate', 'persist-result'],
    'export-warehouse': ['load-context', 'export'],
    'cleanup-stale': ['load-context', 'delete-stale'],
  };
  return byFn[seed.fn.slug] ?? ['load-context', 'call-api', 'persist-result'];
}

const FIRST = ['Ada', 'Leo', 'Mia', 'Sam', 'Ivy', 'Noah', 'Zoe', 'Ken'];
const LAST = ['Lovelace', 'Turing', 'Hopper', 'Chen', 'Patel', 'Kim', 'Diaz'];
const DOMAINS = ['acme.com', 'example.io', 'globex.co', 'initech.dev'];

function person(rng: Rng) {
  const first = rng.pick(FIRST);
  const last = rng.pick(LAST);
  return {
    name: `${first} ${last}`,
    email: `${first.toLowerCase()}@${rng.pick(DOMAINS)}`,
  };
}

/** The data body of the triggering event, chosen by function. */
export function eventData(seed: RunSeed): Json {
  const rng = new Rng('evtdata:' + seed.id);
  const { externalID } = seed.app;
  const { slug } = seed.fn;
  const p = person(rng);

  if (externalID === 'billing') {
    const amount = rng.int(1200, 48000) / 100;
    return {
      invoiceId: `in_${seededULID('inv:' + seed.id, seed.queuedAt).slice(
        0,
        18,
      )}`,
      customerId: `cus_${seededULID('cus:' + seed.id, seed.queuedAt).slice(
        0,
        14,
      )}`,
      amount,
      currency: 'usd',
      status:
        slug === 'retry-failed-charge' ? 'requires_payment_method' : 'paid',
      email: p.email,
    };
  }
  if (externalID === 'email-engine') {
    return {
      userId: `user_${rng.int(10000, 99999)}`,
      email: p.email,
      name: p.name,
      template: rng.pick(['welcome', 'weekly-digest', 'reactivation']),
      locale: rng.pick(['en-US', 'en-GB', 'de-DE']),
    };
  }
  if (externalID === 'user-workflows') {
    return {
      userId: `user_${rng.int(10000, 99999)}`,
      email: p.email,
      name: p.name,
      plan: rng.pick(['free', 'pro', 'enterprise']),
      company: rng.pick(['Acme', 'Globex', 'Initech', 'Umbrella']),
    };
  }
  // data-pipeline
  return {
    batchId: `batch_${seededULID('batch:' + seed.id, seed.queuedAt).slice(
      0,
      16,
    )}`,
    rows: rng.int(500, 25000),
    source: rng.pick(['kafka', 's3', 'postgres-cdc', 'segment']),
    partition: rng.int(0, 15),
  };
}

/** Full Inngest-style event envelope for the trigger panel. */
export function eventEnvelope(seed: RunSeed): Json {
  const name =
    seed.fn.trigger.type === 'EVENT'
      ? seed.fn.trigger.value
      : `${seed.app.externalID}/${seed.fn.slug}.scheduled`;
  return {
    name,
    id: seededULID('evt:' + seed.id, seed.queuedAt),
    ts: seed.queuedAt,
    data: eventData(seed),
    ...(seed.fn.trigger.type === 'CRON' ? { cron: seed.fn.trigger.value } : {}),
  };
}

/** Realistic output for a named step. */
export function stepOutput(seed: RunSeed, stepName: string): Json {
  const rng = new Rng(`stepout:${seed.id}:${stepName}`);
  if (stepName.startsWith('load-') || stepName === 'validate')
    return { ok: true, records: rng.int(1, 25), cacheHit: rng.bool(0.7) };
  if (stepName === 'charge-card')
    return {
      status: 'succeeded',
      chargeId: `ch_${seededULID('ch:' + seed.id, seed.queuedAt).slice(0, 16)}`,
      amount: (eventData(seed).amount as number) ?? rng.int(10, 400),
    };
  if (stepName.includes('email') || stepName === 'notify')
    return {
      delivered: true,
      channel: rng.pick(['email', 'slack', 'webhook']),
      messageId: `msg_${seededULID(
        'm:' + seed.id + stepName,
        seed.queuedAt,
      ).slice(0, 14)}`,
    };
  if (stepName === 'render-pdf' || stepName === 'render-email')
    return { rendered: true, bytes: rng.int(4000, 90000) };
  if (stepName.includes('persist') || stepName === 'delete-stale')
    return { written: true, rowsAffected: rng.int(1, 120) };
  if (stepName === 'aggregate' || stepName === 'reconcile')
    return { groups: rng.int(3, 40), total: rng.int(100, 25000) };
  if (stepName === 'export')
    return {
      exported: rng.int(500, 25000),
      destination: rng.pick(['s3', 'bigquery', 'snowflake']),
    };
  if (
    stepName === 'sync-crm' ||
    stepName === 'provision' ||
    stepName === 'revoke-access'
  )
    return { ok: true, provider: rng.pick(['salesforce', 'hubspot', 'okta']) };
  return {
    status: 200,
    durationMs: rng.int(40, 600),
    body: {
      id: seededULID('api:' + seed.id + stepName, seed.queuedAt).slice(0, 20),
    },
  };
}

/** The function's overall return value (root output) for a completed run. */
export function runOutput(seed: RunSeed): Json {
  const rng = new Rng('runout:' + seed.id);
  const { externalID } = seed.app;
  if (externalID === 'billing') {
    return {
      processed: true,
      invoiceId: eventData(seed).invoiceId,
      amount: eventData(seed).amount,
    };
  }
  if (externalID === 'email-engine') {
    return {
      sent: true,
      messageId: `msg_${seededULID('msg:' + seed.id, seed.queuedAt).slice(
        0,
        16,
      )}`,
    };
  }
  if (externalID === 'user-workflows') {
    return { synced: true, steps: rng.int(2, 5) };
  }
  return { rowsProcessed: rng.int(500, 25000), durationMs: rng.int(800, 9000) };
}

const ERRORS = [
  {
    name: 'TimeoutError',
    message: 'Upstream request timed out after 30000ms',
    cause: 'ETIMEDOUT connecting to api.stripe.com:443',
  },
  {
    name: 'ValidationError',
    message: 'Invalid payload: missing required field "customerId"',
    cause: null,
  },
  {
    name: 'HTTPError',
    message: 'Request failed with status code 502',
    cause: 'Bad Gateway from downstream service',
  },
  {
    name: 'RateLimitError',
    message: 'Too many requests: rate limit exceeded (429)',
    cause: 'Retry after 12s',
  },
];

/** A deterministic error object for a failed run/step. */
export function runError(seed: RunSeed): Json {
  const rng = new Rng('runerr:' + seed.id);
  const e = rng.pick(ERRORS);
  return {
    name: e.name,
    message: e.message,
    cause: e.cause,
    stack: `${e.name}: ${e.message}\n    at ${seed.fn.slug} (/app/functions/${
      seed.fn.slug
    }.ts:${rng.int(12, 180)}:${rng.int(
      3,
      40,
    )})\n    at process.processTicksAndRejections (node:internal/process/task_queues:95:5)`,
  };
}
