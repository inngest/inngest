export type Trace = {
  attempts: number | null;
  childrenSpans?: Trace[];
  endedAt: string | null;
  isRoot: boolean;
  name: string;
  outputID: string | null;
  queuedAt: string;
  spanID: string;
  stepID?: string | null;
  startedAt: string | null;
  status: string;
  stepInfo: StepInfoInvoke | StepInfoSleep | StepInfoWait | StepInfoRun | StepInfoSignal | null;
  stepOp?: string | null;
  stepType?: string | null;
  userlandSpan: UserlandSpanType | null;
  isUserland: boolean;
  debugRunID?: string | null;
  debugSessionID?: string | null;
  metadata?: SpanMetadata[];
};

export type SpanMetadataKind =
  | `inngest.http`
  | `inngest.ai`
  | `inngest.warnings`
  | SpanMetadataKindUserland;

export type SpanMetadataKindUserland = `userland.${string}`;

export type SpanMetadataScope = 'run' | 'step' | 'step_attempt' | 'extended_trace';

export type SpanMetadata =
  | SpanMetadataInngestAI
  | SpanMetadataInngestHTTP
  | SpanMetadataInngestWarnings
  | SpanMetadataUserland
  | SpanMetadataUnknown;

export type SpanMetadataInngestAI = {
  scope: 'step_attempt' | 'extended_trace';
  kind: 'inngest.ai';
  updatedAt: string;
  values: {
    input_tokens?: number;
    output_tokens?: number;
    model: string;
    system: string;
    operation_name: string;
  };
};

export type SpanMetadataInngestHTTP = {
  scope: 'extended_trace';
  kind: 'inngest.http';
  updatedAt: string;
  values: {
    method: string;
    domain: string;
    path: string;
    request_size?: number;
    request_content_type?: string;
    response_size?: number;
    response_status?: number;
    response_content_type?: string;
  };
};

export type SpanMetadataInngestWarnings = {
  scope: SpanMetadataScope;
  kind: 'inngest.warnings';
  updatedAt: string;
  values: Record<string, string>;
};

export type SpanMetadataUserland = {
  scope: SpanMetadataScope;
  kind: SpanMetadataKindUserland;
  updatedAt: string;
  values: Record<string, unknown>;
};

export type SpanMetadataUnknown = {
  scope: SpanMetadataScope;
  kind: SpanMetadataKind;
  updatedAt: string;
  values: Record<string, unknown>;
};

export type UserlandSpanType = {
  spanName: string | null;
  spanKind: string | null;
  serviceName: string | null;
  scopeName: string | null;
  scopeVersion: string | null;
  spanAttrs: string | null;
  resourceAttrs: string | null;
};

export type StepInfoInvoke = {
  triggeringEventID: string;
  functionID: string;
  timeout: string;
  returnEventID: string | null;
  runID: string | null;
  timedOut: boolean | null;
};

export type StepInfoSleep = {
  sleepUntil: string;
};

export type StepInfoWait = {
  eventName: string;
  expression: string | null;
  timeout: string;
  foundEventID: string | null;
  timedOut: boolean | null;
};

export type StepInfoRun = {
  type: string | null;
};

export type StepInfoSignal = {
  signal: string;
  timeout: string;
  timedOut: boolean | null;
};

export function isStepInfoRun(stepInfo: Trace['stepInfo']): stepInfo is StepInfoRun {
  if (!stepInfo) {
    return false;
  }

  return 'type' in stepInfo;
}

export function isStepInfoInvoke(stepInfo: Trace['stepInfo']): stepInfo is StepInfoInvoke {
  if (!stepInfo) {
    return false;
  }

  return 'triggeringEventID' in stepInfo;
}

export function isStepInfoSleep(stepInfo: Trace['stepInfo']): stepInfo is StepInfoSleep {
  if (!stepInfo) {
    return false;
  }

  return 'sleepUntil' in stepInfo;
}

export function isStepInfoWait(stepInfo: Trace['stepInfo']): stepInfo is StepInfoWait {
  if (!stepInfo) {
    return false;
  }

  return 'foundEventID' in stepInfo;
}

export function isStepInfoSignal(stepInfo: Trace['stepInfo']): stepInfo is StepInfoSignal {
  if (!stepInfo) {
    return false;
  }

  return 'signal' in stepInfo;
}

// ============================================================================
// Timing Breakdown Types (EXE-1217)
// ============================================================================

/**
 * Timing breakdown categories for span visualization
 * Note: 'customer_server' displays as "{ORG_NAME} SERVER" in UI
 * Note: 'connecting' is reserved for Phase 2
 */
export type TimingCategory = 'inngest' | 'connecting' | 'customer_server';

/**
 * Segment types within each category
 */
export type InngestSegmentType = 'queue' | 'concurrency_delay' | 'processing';
export type ConnectingSegmentType = 'request' | 'handshake';
export type CustomerServerSegmentType = 'middleware' | 'running' | 'db_query' | 'checkpointing';

export type SegmentType = InngestSegmentType | ConnectingSegmentType | CustomerServerSegmentType;

/**
 * Individual timing segment within a category
 */
export type TimingSegment = {
  /** Parent category */
  category: TimingCategory;

  /** Specific segment type */
  segmentType: SegmentType;

  /** Display label (e.g., "Queue", "Running") */
  label: string;

  /** Duration in milliseconds */
  durationMs: number;

  /** Tailwind background class */
  color: string;

  /** If true, render with striped/hatched pattern (for waiting states) */
  isWaiting?: boolean;

  /** Nested segments (e.g., DB Query within Running) - Phase 2 */
  children?: TimingSegment[];
};

/**
 * Category totals for the breakdown panel
 */
export type TimingCategoryTotal = {
  /** Category identifier */
  category: TimingCategory;

  /** Display label (e.g., "INNGEST", "ACME CORP SERVER") */
  label: string;

  /** Icon identifier for rendering */
  icon: 'gear' | 'lightning' | 'building';

  /** Total milliseconds for this category */
  totalMs: number;

  /** Individual segments within this category */
  segments: TimingSegment[];
};

/**
 * Complete timing breakdown for a span
 */
export type SpanTimingBreakdown = {
  /** Total duration from queuedAt to endedAt */
  totalDurationMs: number;

  /** Category breakdowns with segment details */
  categories: TimingCategoryTotal[];

  /** For the top segmented bar rendering */
  barSegments: Array<{
    category: TimingCategory;
    widthPercent: number;
    isWaiting?: boolean;
  }>;

  /** Raw timestamps (epoch ms) */
  queuedAt: number;
  startedAt: number | null;
  endedAt: number | null;

  /** Formatted display timestamps */
  startTime: string;
  endTime: string;
};
