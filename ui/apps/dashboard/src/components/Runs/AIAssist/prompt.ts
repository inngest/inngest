import { DEFAULT_TIME } from '@inngest/components/hooks/useCalculatedStartTime';
import {
  parseDuration,
  subtractDuration,
} from '@inngest/components/utils/date';

export type RunPromptContext = {
  account: string;
  environmentSlug: string;
  scope: 'env' | 'fn';
  apps?: ReadonlyArray<{ id: string; externalID: string }>;
  functionAppID?: string;
  functionSlug?: string;
  search: Record<string, unknown>;
};

export type RunPromptResult =
  | { prompt: string; error?: never }
  | { prompt?: never; error: string };

const statuses = new Set([
  'QUEUED',
  'RUNNING',
  'COMPLETED',
  'FAILED',
  'CANCELLED',
  'SKIPPED',
]);
const timeFields: Record<string, string> = {
  QUEUED_AT: 'queuedAt',
  STARTED_AT: 'startedAt',
  ENDED_AT: 'endedAt',
};
const supportedFilterKeys = new Set(['filterApp', 'filterStatus']);

export function buildRunDiscoveryPrompt(
  context: RunPromptContext,
  copiedAt: Date,
): RunPromptResult {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(context.search)) {
    if (value !== undefined && value !== null) params.set(key, String(value));
  }
  let selectedStatuses: string[];
  let selectedApps: string[];
  try {
    selectedStatuses = parseSelection(context.search.filterStatus);
    selectedApps = parseSelection(context.search.filterApp);
  } catch {
    return { error: 'Select valid app and status filters to copy a prompt.' };
  }
  const blocked: string[] = [];
  if (params.get('search')) blocked.push('CEL search');
  if (params.get('excludeDeferred') === 'true') blocked.push('Run type');
  if (selectedStatuses.includes('PAUSED')) blocked.push('PAUSED status');
  if (
    selectedStatuses.some(
      (status) => status !== 'PAUSED' && !statuses.has(status),
    )
  ) {
    blocked.push('unsupported status');
  }
  for (const [key, value] of params) {
    if (value && key.startsWith('filter') && !supportedFilterKeys.has(key)) {
      blocked.push('additional filters');
    }
  }
  if (blocked.length) {
    return {
      error: `AI prompts currently support basic filters. Clear ${[...new Set(blocked)].join(', ')} to copy a prompt.`,
    };
  }
  if (!context.account || !context.environmentSlug) {
    return {
      error:
        'Unable to resolve the account or environment. Reload this page and try again.',
    };
  }

  const timeField = timeFields[params.get('timeField') ?? 'QUEUED_AT'];
  if (!timeField)
    return { error: 'Select a supported time field to copy a prompt.' };

  let from: Date;
  let until: Date;
  try {
    const relative = params.get('last');
    const start = params.get('start');
    from = relative
      ? subtractDuration(copiedAt, parseDuration(relative))
      : start
        ? new Date(start)
        : subtractDuration(copiedAt, parseDuration(DEFAULT_TIME));
    until = params.get('end') ? new Date(params.get('end')!) : copiedAt;
    if (
      !Number.isFinite(from.getTime()) ||
      !Number.isFinite(until.getTime()) ||
      from >= until
    ) {
      return { error: 'Select a valid time range to copy a prompt.' };
    }
  } catch {
    return { error: 'Select a valid time range to copy a prompt.' };
  }

  const appIDs: string[] = [];
  let functionID: string | undefined;
  if (context.scope === 'fn') {
    if (!context.functionAppID || !context.functionSlug) {
      return {
        error: 'Unable to resolve this function. Retry loading its metadata.',
      };
    }
    appIDs.push(context.functionAppID);
    // Match the public function ID used by the Runs REST adapter.
    const prefix = `${context.functionAppID}-`;
    functionID = context.functionSlug.startsWith(prefix)
      ? context.functionSlug.slice(prefix.length)
      : context.functionSlug;
    if (!functionID)
      return {
        error: 'Unable to resolve this function. Retry loading its metadata.',
      };
  } else {
    for (const selectedID of selectedApps) {
      const app = context.apps?.find(
        (item) => item.id === selectedID || item.externalID === selectedID,
      );
      if (!app)
        return {
          error:
            'Unable to resolve a selected app. Retry loading its metadata.',
        };
      appIDs.push(app.externalID);
    }
  }

  const statusDescription = selectedStatuses.length
    ? `${selectedStatuses.map((status) => status.toLowerCase()).join(' or ')} `
    : '';
  const functionDescription = functionID ? ` for ${quote(functionID)}` : '';
  const uniqueApps = [...new Set(appIDs)];
  const appDescription = uniqueApps.length
    ? ` in ${uniqueApps.length === 1 ? 'app' : 'apps'} ${uniqueApps.map(quote).join(' or ')}`
    : '';
  const relative = !params.get('end')
    ? (params.get('last') ?? (!params.get('start') ? DEFAULT_TIME : undefined))
    : undefined;
  const range = relative
    ? `the last ${describeDuration(relative)}`
    : `${from.toISOString()} and ${until.toISOString()}`;
  const timeDescription =
    timeField === 'queuedAt'
      ? `${relative ? 'from' : 'between'} ${range}`
      : `that ${timeField === 'startedAt' ? 'started' : 'ended'} ${relative ? 'in' : 'between'} ${range}`;
  const latest =
    selectedStatuses.length === 1 && selectedStatuses[0] === 'FAILED'
      ? 'failure'
      : 'run';
  return {
    prompt: `Find ${statusDescription}runs${functionDescription}${appDescription} in ${quote(context.environmentSlug)} ${timeDescription}. Show the run IDs and inspect the trace of the most recent ${latest}. Use the ${quote(context.account)} account.`,
  };
}

// The dashboard stores arrays as JSON strings in parsed router search state.
function parseSelection(value: unknown): string[] {
  if (value === undefined || value === null) return [];
  if (typeof value !== 'string') throw new Error('Invalid selection');
  const parsed: unknown = JSON.parse(value);
  if (
    !Array.isArray(parsed) ||
    !parsed.every((item) => typeof item === 'string')
  ) {
    throw new Error('Invalid selection');
  }
  return parsed;
}

function quote(value: string): string {
  return `“${value.replaceAll('“', '\\u201c').replaceAll('”', '\\u201d')}”`;
}

function describeDuration(value: string): string {
  const duration = parseDuration(value);
  if (duration.days === 1) return '24 hours';
  return Object.entries(duration)
    .map(
      ([unit, count]) => `${count} ${count === 1 ? unit.slice(0, -1) : unit}`,
    )
    .join(' ');
}
