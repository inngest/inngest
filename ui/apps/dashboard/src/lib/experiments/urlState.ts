import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';
import { subtractDuration } from '@inngest/components/utils/date';

type TimeRangePreset = '24h' | '7d' | '30d';

export const EXPERIMENT_DEFAULT_TIME_PRESET = '24h' satisfies TimeRangePreset;

export type ExperimentDetailPanel = 'info' | 'scoring' | 'none';

export type ExperimentLiveTimeRange = {
  type: 'live';
  durationMs: number;
  preset: TimeRangePreset | null;
};

export type ExperimentFixedTimeRange = {
  type: 'fixed';
  fromTs: number;
  toTs: number;
  preset: null;
};

export type ExperimentTimeRange =
  | ExperimentLiveTimeRange
  | ExperimentFixedTimeRange;

export type ExperimentDetailSearchParams = {
  from_ts?: number;
  to_ts?: number;
  live?: boolean;
  tpl_var_variant?: string | string[];
  show_inactive?: boolean;
  panel?: ExperimentDetailPanel;
  score_formula?: string;
};

export type ExperimentScoringFormula = {
  metrics: ExperimentScoringFormulaMetric[];
};

export type ExperimentScoringFormulaMetric = Pick<
  ExperimentScoringMetric,
  'key'
> &
  Partial<Omit<ExperimentScoringMetric, 'key'>>;

export type ExperimentUrlState = {
  timeRange: ExperimentTimeRange;
  selectedVariants: string[];
  showInactive: boolean;
  panel: ExperimentDetailPanel;
  scoreFormula: ExperimentScoringFormula | null;
  scoreFormulaParam?: string;
};

const TIME_PRESET_DURATIONS_MS: Record<TimeRangePreset, number> = {
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
  '30d': 30 * 24 * 60 * 60 * 1000,
};

const SECOND_MS = 1000;
const MINUTE_MS = 60 * SECOND_MS;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;
const PRESET_MATCH_TOLERANCE_MS = 60 * 1000;
const SCORE_FORMULA_VERSION_PREFIX = 'v1:';

export function isTimeRangePreset(value: unknown): value is TimeRangePreset {
  return value === '24h' || value === '7d' || value === '30d';
}

export function validateExperimentDetailSearch(
  search: Record<string, unknown>,
): ExperimentDetailSearchParams {
  return {
    from_ts: readTimestamp(search.from_ts),
    to_ts: readTimestamp(search.to_ts),
    live: readBoolean(search.live),
    tpl_var_variant: readStringOrStringArray(search.tpl_var_variant),
    show_inactive: readBoolean(search.show_inactive),
    panel: readPanel(search.panel),
    score_formula: readString(search.score_formula),
  };
}

export function getExperimentUrlState(
  search: ExperimentDetailSearchParams,
): ExperimentUrlState {
  return {
    timeRange: getExperimentTimeRange(search),
    selectedVariants: parseVariantTemplateVariable(search.tpl_var_variant),
    showInactive: search.show_inactive === true,
    panel: search.panel ?? 'info',
    scoreFormula: parseExperimentScoringFormula(search.score_formula),
    scoreFormulaParam: search.score_formula,
  };
}

export function getExperimentTimeRangeDates(
  timeRange: ExperimentTimeRange,
  now = new Date(),
): { from: Date; to: Date } {
  if (timeRange.type === 'fixed') {
    return {
      from: new Date(timeRange.fromTs),
      to: new Date(timeRange.toTs),
    };
  }

  return {
    from: new Date(now.getTime() - timeRange.durationMs),
    to: now,
  };
}

export function experimentTimeRangeToRangeChange(
  timeRange: ExperimentTimeRange,
): RangeChangeProps {
  if (timeRange.type === 'fixed') {
    return {
      type: 'absolute',
      start: new Date(timeRange.fromTs),
      end: new Date(timeRange.toTs),
    };
  }

  return {
    type: 'relative',
    duration: durationMsToDuration(timeRange.durationMs),
  };
}

export function hasExperimentTimeRangeSearch(
  search: ExperimentDetailSearchParams,
): boolean {
  return (
    search.from_ts !== undefined &&
    search.to_ts !== undefined &&
    search.to_ts > search.from_ts
  );
}

export function getExperimentTimeRangeSearch(
  preset: TimeRangePreset,
  now = Date.now(),
): Record<string, unknown> {
  if (preset === EXPERIMENT_DEFAULT_TIME_PRESET) {
    return {
      from_ts: undefined,
      to_ts: undefined,
      live: undefined,
    };
  }

  const durationMs = TIME_PRESET_DURATIONS_MS[preset];
  return {
    from_ts: now - durationMs,
    to_ts: now,
    live: true,
  };
}

export function setExperimentTimeRangeSearch(
  prev: Record<string, unknown>,
  range: TimeRangePreset | RangeChangeProps,
  now = Date.now(),
): Record<string, unknown> {
  const next = { ...prev };
  const timeSearch =
    typeof range === 'string'
      ? getExperimentTimeRangeSearch(range, now)
      : getRangeTimeRangeSearch(range, now);
  applyOptionalParam(next, 'from_ts', timeSearch.from_ts);
  applyOptionalParam(next, 'to_ts', timeSearch.to_ts);
  applyOptionalParam(next, 'live', timeSearch.live);
  return next;
}

export function setExperimentVariantsSearch(
  prev: Record<string, unknown>,
  variants: string[],
): Record<string, unknown> {
  const next = { ...prev };
  applyOptionalParam(
    next,
    'tpl_var_variant',
    variants.length > 0
      ? serializeVariantTemplateVariable(variants)
      : undefined,
  );
  return next;
}

export function setExperimentShowInactiveSearch(
  prev: Record<string, unknown>,
  showInactive: boolean,
): Record<string, unknown> {
  const next = { ...prev };
  applyOptionalParam(next, 'show_inactive', showInactive ? true : undefined);
  return next;
}

export function setExperimentPanelSearch(
  prev: Record<string, unknown>,
  panel: ExperimentDetailPanel,
): Record<string, unknown> {
  const next = { ...prev };
  applyOptionalParam(next, 'panel', panel === 'info' ? undefined : panel);
  return next;
}

export function setExperimentScoringFormulaSearch(
  prev: Record<string, unknown>,
  formulaParam: string | undefined,
): Record<string, unknown> {
  const next = { ...prev };
  applyOptionalParam(next, 'score_formula', formulaParam);
  return next;
}

export function serializeExperimentScoringFormula(
  metrics: ExperimentScoringFormulaMetric[],
): string {
  return `${SCORE_FORMULA_VERSION_PREFIX}${JSON.stringify({
    m: metrics.map(serializeFormulaMetric),
  })}`;
}

export function parseExperimentScoringFormula(
  value: string | undefined,
): ExperimentScoringFormula | null {
  if (!value?.startsWith(SCORE_FORMULA_VERSION_PREFIX)) return null;

  let parsed: unknown;
  try {
    parsed = JSON.parse(value.slice(SCORE_FORMULA_VERSION_PREFIX.length));
  } catch {
    return null;
  }

  if (!isRecord(parsed) || !Array.isArray(parsed.m)) return null;

  const metrics: ExperimentScoringFormulaMetric[] = [];
  for (const item of parsed.m) {
    const metric = parseFormulaMetric(item);
    if (metric) metrics.push(metric);
  }

  return { metrics };
}

export function serializeVariantTemplateVariable(variants: string[]): string {
  return variants.map(escapeTemplateVariableValue).join(',');
}

export function parseVariantTemplateVariable(
  value: string | string[] | undefined,
): string[] {
  if (value === undefined) return [];

  const values = Array.isArray(value) ? value : splitEscapedList(value);
  return values.filter((v) => v.length > 0);
}

function getExperimentTimeRange(
  search: ExperimentDetailSearchParams,
): ExperimentTimeRange {
  const fromTs = search.from_ts;
  const toTs = search.to_ts;

  if (fromTs === undefined || toTs === undefined || toTs <= fromTs) {
    return liveTimeRangeForPreset(EXPERIMENT_DEFAULT_TIME_PRESET);
  }

  const durationMs = toTs - fromTs;
  if (search.live === true) {
    return {
      type: 'live',
      durationMs,
      preset: presetForDuration(durationMs),
    };
  }

  return {
    type: 'fixed',
    fromTs,
    toTs,
    preset: null,
  };
}

function liveTimeRangeForPreset(
  preset: TimeRangePreset,
): ExperimentLiveTimeRange {
  return {
    type: 'live',
    durationMs: TIME_PRESET_DURATIONS_MS[preset],
    preset,
  };
}

function presetForDuration(durationMs: number): TimeRangePreset | null {
  for (const preset of Object.keys(
    TIME_PRESET_DURATIONS_MS,
  ) as TimeRangePreset[]) {
    const expected = TIME_PRESET_DURATIONS_MS[preset];
    if (Math.abs(durationMs - expected) <= PRESET_MATCH_TOLERANCE_MS) {
      return preset;
    }
  }

  return null;
}

function getRangeTimeRangeSearch(
  range: RangeChangeProps,
  now: number,
): Record<string, unknown> {
  if (range.type === 'absolute') {
    return {
      from_ts: range.start.getTime(),
      to_ts: range.end.getTime(),
      live: undefined,
    };
  }

  const to = new Date(now);
  return {
    from_ts: subtractDuration(to, range.duration).getTime(),
    to_ts: now,
    live: true,
  };
}

function durationMsToDuration(
  durationMs: number,
): Extract<RangeChangeProps, { type: 'relative' }>['duration'] {
  if (durationMs % DAY_MS === 0) return { days: durationMs / DAY_MS };
  if (durationMs % HOUR_MS === 0) return { hours: durationMs / HOUR_MS };
  if (durationMs % MINUTE_MS === 0) {
    return { minutes: durationMs / MINUTE_MS };
  }
  return { seconds: Math.max(1, Math.round(durationMs / SECOND_MS)) };
}

function readTimestamp(value: unknown): number | undefined {
  if (typeof value !== 'string' && typeof value !== 'number') return undefined;

  const numberValue = Number(value);
  return Number.isFinite(numberValue) && numberValue > 0
    ? Math.trunc(numberValue)
    : undefined;
}

function readBoolean(value: unknown): boolean | undefined {
  if (typeof value === 'boolean') return value;
  if (value === 'true') return true;
  if (value === 'false') return false;
  return undefined;
}

function readPanel(value: unknown): ExperimentDetailPanel | undefined {
  return value === 'info' || value === 'scoring' || value === 'none'
    ? value
    : undefined;
}

function readString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined;
}

function readStringOrStringArray(
  value: unknown,
): string | string[] | undefined {
  if (typeof value === 'string') return value;
  if (Array.isArray(value)) {
    const strings = value.filter((v): v is string => typeof v === 'string');
    return strings.length > 0 ? strings : undefined;
  }
  return undefined;
}

function applyOptionalParam(
  search: Record<string, unknown>,
  key: string,
  value: unknown,
) {
  if (value === undefined || value === null || value === false) {
    delete search[key];
    return;
  }

  search[key] = value;
}

function escapeTemplateVariableValue(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/,/g, '\\,');
}

function splitEscapedList(value: string): string[] {
  const values: string[] = [];
  let current = '';
  let escaping = false;

  for (const char of value) {
    if (escaping) {
      current += char;
      escaping = false;
      continue;
    }

    if (char === '\\') {
      escaping = true;
      continue;
    }

    if (char === ',') {
      values.push(current);
      current = '';
      continue;
    }

    current += char;
  }

  if (escaping) current += '\\';
  values.push(current);
  return values;
}

function serializeFormulaMetric(
  metric: ExperimentScoringFormulaMetric,
): Record<string, unknown> {
  const item: Record<string, unknown> = { k: metric.key };

  if (metric.enabled !== undefined) item.e = metric.enabled;
  if (metric.points !== undefined) item.p = metric.points;
  if (metric.minValue !== undefined) item.min = metric.minValue;
  if (metric.maxValue !== undefined) item.max = metric.maxValue;
  if (metric.invert !== undefined) item.inv = metric.invert;
  if (metric.labelBest !== undefined) item.best = metric.labelBest;
  if (metric.labelWorst !== undefined) item.worst = metric.labelWorst;
  if (metric.displayName !== undefined) item.name = metric.displayName;

  return item;
}

function parseFormulaMetric(
  value: unknown,
): ExperimentScoringFormulaMetric | null {
  if (!isRecord(value)) return null;

  const key = readString(value.k);
  if (!key) return null;

  const enabled = readBoolean(value.e);
  const points = readFormulaNumber(value.p);
  const minValue = readFormulaNumber(value.min);
  const maxValue = readFormulaNumber(value.max);
  const invert = readBoolean(value.inv);
  const labelBest = readString(value.best);
  const labelWorst = readString(value.worst);
  const displayName = readString(value.name);

  const metric: ExperimentScoringFormulaMetric = { key };
  if (enabled !== undefined) metric.enabled = enabled;
  if (points !== undefined) metric.points = Math.max(0, Math.round(points));
  if (minValue !== undefined) metric.minValue = minValue;
  if (maxValue !== undefined) metric.maxValue = maxValue;
  if (invert !== undefined) metric.invert = invert;
  if (labelBest !== undefined) metric.labelBest = labelBest;
  if (labelWorst !== undefined) metric.labelWorst = labelWorst;
  if (displayName !== undefined) metric.displayName = displayName;

  return Object.keys(metric).length > 1 ? metric : null;
}

function readFormulaNumber(value: unknown): number | undefined {
  if (typeof value !== 'number' || !Number.isFinite(value)) return undefined;
  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
