import { isScoreKind } from '../RunDetails/ScoresAttrs';
import {
  KindInngestAISummary,
  KindInngestSandbox,
  KindInngestWarnings,
  KindPrefixInngestWarning,
  type AIMetadata,
  type AISummaryMetadata,
  type SpanMetadataKind as GeneratedSpanMetadataKind,
  type SpanMetadataKindInngestScore as GeneratedSpanMetadataKindInngestScore,
  type SpanMetadataKindInngestWarnings as GeneratedSpanMetadataKindInngestWarnings,
  type SpanMetadataKindUserland as GeneratedSpanMetadataKindUserland,
  type SandboxMetadata,
  type Warnings,
} from '../generated/index';

export type Trace = {
  attempts: number | null;
  childrenSpans?: Trace[];
  endedAt: string | null;
  isRoot: boolean;
  name: string;
  outputID: string | null;
  queuedAt: string;
  scheduledAt: string | null;
  spanID: string;
  stepID?: string | null;
  groupID?: string | null;
  startedAt: string | null;
  status: string;
  stepInfo: StepInfoInvoke | StepInfoSleep | StepInfoWait | StepInfoRun | StepInfoSignal | null;
  stepOp?: string | null;
  stepType?: string | null;
  /** A span group's kind, as its caller named it, like `job` or `agent` */
  groupKind?: string | null;
  /** The library that created this step or span group, as `<package>@<version>` */
  origin?: string | null;
  userlandSpan: UserlandSpanType | null;
  isUserland: boolean;
  debugRunID?: string | null;
  debugSessionID?: string | null;
  metadata?: SpanMetadata[];
  response?: ResponseInfo;
};

export type ResponseInfo = {
  statusCode: number;
  headers: Record<string, string | string[]>;
};

export type SpanMetadataKind = GeneratedSpanMetadataKind;

export type SpanMetadataKindInngestScore = GeneratedSpanMetadataKindInngestScore;

export type SpanMetadataKindInngestWarnings = GeneratedSpanMetadataKindInngestWarnings;

export type SpanMetadataKindUserland = GeneratedSpanMetadataKindUserland;

export type SpanMetadataScope = 'run' | 'step' | 'step_attempt' | 'extended_trace';

export type SpanMetadata =
  | SpanMetadataInngestAI
  | SpanMetadataInngestAISummary
  | SpanMetadataInngestExperiment
  | SpanMetadataInngestHTTP
  | SpanMetadataInngestHTTPTiming
  | SpanMetadataInngestTiming
  | SpanMetadataInngestResponseHeaders
  | SpanMetadataInngestSandbox
  | SpanMetadataInngestScore
  | SpanMetadataInngestWarnings
  | SpanMetadataUserland
  | SpanMetadataUnknown;

export type SpanMetadataInngestAI = {
  scope: 'step_attempt' | 'extended_trace';
  kind: 'inngest.ai';
  updatedAt: string;
  values: AIMetadata;
};

// The run-level AI usage rollup synthesized by the backend on every span-tree
// read; it only ever appears on the root span.
export type SpanMetadataInngestAISummary = {
  scope: 'run';
  kind: typeof KindInngestAISummary;
  updatedAt: string;
  values: AISummaryMetadata;
};

export type SpanMetadataInngestExperiment = {
  scope: SpanMetadataScope;
  kind: 'inngest.experiment';
  updatedAt: string;
  values: {
    name: string;
    experiment_name?: string; // TODO: remove this in a month or so
    variant: string;
    selection_strategy: string;
    available_variants?: string[];
    variant_weights?: Record<string, number>;
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

export type SpanMetadataInngestHTTPTiming = {
  scope: 'step_attempt';
  kind: 'inngest.http.timing';
  updatedAt: string;
  values: {
    dns_lookup_ms: number;
    tcp_connection_ms: number;
    tls_handshake_ms: number;
    server_processing_ms: number;
    content_transfer_ms: number;
    total_ms: number;
  };
};

export type SpanMetadataInngestTiming = {
  scope: 'step_attempt';
  kind: 'inngest.timing';
  updatedAt: string;
  values: {
    queue_delay_ms?: number;
    system_latency_ms?: number;
    network_total_ms?: number;
    total_inngest_ms?: number;
  };
};

export type SpanMetadataInngestResponseHeaders = {
  scope: 'extended_trace' | 'step_attempt';
  kind: 'inngest.response_headers';
  updatedAt: string;
  values: Record<string, string>;
};

export type SpanMetadataInngestWarnings = {
  scope: SpanMetadataScope;
  kind: SpanMetadataKindInngestWarnings;
  updatedAt: string;
  values: Warnings;
};

export type SpanMetadataInngestSandbox = {
  scope: SpanMetadataScope;
  kind: typeof KindInngestSandbox;
  updatedAt: string;
  values: SandboxMetadata;
};

export type SpanMetadataInngestScore = {
  scope: SpanMetadataScope;
  kind: SpanMetadataKindInngestScore;
  updatedAt: string;
  // `inngest.score.<name>` holds the score's `{value}`, the legacy
  // `inngest.score` maps each score name to its `{value}`. Read these through
  // scoreRows, which handles both.
  values: { value: number | boolean } | Record<string, { value: number | boolean }>;
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

export function isExperimentMetadata(md: SpanMetadata): md is SpanMetadataInngestExperiment {
  return md.kind === 'inngest.experiment';
}

export function isSandboxMetadata(md: SpanMetadata): md is SpanMetadataInngestSandbox {
  return md.kind === KindInngestSandbox;
}

/**
 * True for both warning storage forms: the legacy merged `inngest.warnings`
 * and the per-code `inngest.warning.<code>`. Unrelated kinds such as
 * `inngest.warningsfoo` do not match.
 */
export function isWarningMetadata(md: { kind: string }): boolean {
  return md.kind === 'inngest.warnings' || md.kind.startsWith('inngest.warning.');
}

export function isScoreMetadata(md: SpanMetadata): md is SpanMetadataInngestScore {
  return isScoreKind(md.kind);
}

// Matches the legacy `inngest.warnings` kind (a map of code to message) and per
// code `inngest.warning.<code>` kinds.
export function isWarningMetadata(md: SpanMetadata): md is SpanMetadataInngestWarnings {
  return (
    md.kind === KindInngestWarnings ||
    (md.kind.startsWith(KindPrefixInngestWarning) &&
      md.kind.length > KindPrefixInngestWarning.length)
  );
}

export function isAISummaryMetadata(md: SpanMetadata): md is SpanMetadataInngestAISummary {
  return md.kind === KindInngestAISummary;
}

/**
 * Whether a span is a virtual span group: the steps an SDK called inside a
 * span, nested by the API.
 */
export function isSpanGroup(trace: Pick<Trace, 'stepType'>): boolean {
  return trace.stepType === 'SPAN_GROUP';
}
