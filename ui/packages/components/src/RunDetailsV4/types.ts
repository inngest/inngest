import type {
  SpanMetadataKind as GeneratedSpanMetadataKind,
  SpanMetadataKindUserland as GeneratedSpanMetadataKindUserland,
  Warnings,
} from '../generated/index';

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

export type SpanMetadataKind = GeneratedSpanMetadataKind;

export type SpanMetadataKindUserland = GeneratedSpanMetadataKindUserland;

export type SpanMetadataScope = 'run' | 'step' | 'step_attempt' | 'extended_trace';

export type SpanMetadata =
  | SpanMetadataInngestAI
  | SpanMetadataInngestHTTP
  | SpanMetadataInngestHTTPTiming
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

export type SpanMetadataInngestWarnings = {
  scope: SpanMetadataScope;
  kind: 'inngest.warnings';
  updatedAt: string;
  values: Warnings;
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
