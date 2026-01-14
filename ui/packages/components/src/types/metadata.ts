export type SpanMetadataKind =
  | 'inngest.http'
  | 'inngest.ai'
  | 'inngest.warnings'
  | 'inngest.skip'
  | SpanMetadataKindUserland;

export type SpanMetadataKindUserland = `userland.${string}`;

export type SpanMetadataScope = 'run' | 'step' | 'step_attempt' | 'extended_trace';

export type SpanMetadata =
  | SpanMetadataInngestAI
  | SpanMetadataInngestHTTP
  | SpanMetadataInngestWarnings
  | SpanMetadataInngestSkip
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

export type SpanMetadataInngestSkip = {
  scope: 'run';
  kind: 'inngest.skip';
  updatedAt: string;
  values: {
    reason?: string;
    existing_run_id?: string;
  };
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

export function isSpanMetadataSkip(metadata: SpanMetadata): metadata is SpanMetadataInngestSkip {
  return metadata.kind === 'inngest.skip';
}
