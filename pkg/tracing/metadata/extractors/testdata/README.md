# AI metadata OTLP fixtures

Real OTLP/JSON `ExportTraceServiceRequest` payloads captured from instrumented
AI SDK calls, one directory per instrumentation. Each directory holds one
fixture per call variant (OpenAI: `params_chat`, `tools_chat`, `stream_chat`,
`basic_responses`, `reasoning_responses`, `embeddings`; Anthropic:
`reasoning_messages`) where the instrumentation emits a span for it.

`TestAIMetadataExtractor_CapturedFixtures` extracts `AIMetadata` from every
span of every fixture and asserts against the `<fixture>.out` golden file
alongside it. Regenerate goldens with:

    go test ./pkg/tracing/metadata/extractors -run CapturedFixtures -update

The goldens render empty fields explicitly (`""`, `null`) — what an
instrumentation does *not* emit is part of what they lock in.

## Per-instrumentation notes

### openai_otel_official — `@opentelemetry/instrumentation-openai`

Standard `gen_ai.*` semconv attributes; one span per call. No Responses API
variants (the instrumentation doesn't cover `responses.create`).

### openai_otel_traceloop — `@traceloop/instrumentation-openai`

`gen_ai.*` attributes. Emits OpenAI's native `tool_calls` finish reason as the
singular `tool_call`; finish reasons are stored raw per emitter. Streaming
chat spans carry no usage, so token counts stay zero and `total_tokens` is not
derived.

### openai_langsmith_otel — `langsmith` `wrapOpenAI` in OTel mode (`initializeOTEL`)

Carries a standard `gen_ai.*` set alongside `langsmith.*`/`ls_*` keys, so
extraction works through the semconv mappings with no LangSmith-specific
convention. `model` is the requested alias (`gen_ai.request.model`); the dated
model lands in `response_model`. `gen_ai.response.finish_reasons` arrives as a
scalar string (not the semconv array); the extractor's scalar fallback wraps it
into a single-element list. The Responses API variants omit finish reasons
entirely. No embeddings or
stream_chat fixtures: `wrapOpenAI` (langsmith 0.7.3) wraps neither
`embeddings.create` nor streaming chat completions, so those variants emit no
span.

### anthropic_otel_traceloop — `@traceloop/instrumentation-anthropic`

Anthropic Messages API (`messages.create`), `gen_ai.*` attributes, one span per
call. Like the OpenAI Traceloop emitter it uses the current
`gen_ai.provider.name` (`anthropic`) and emits `gen_ai.usage.total_tokens`
directly (not derived). `response_id` is empty — the instrumentation emits no
`gen_ai.response.id`. The single `reasoning_messages` fixture uses adaptive
extended thinking: Anthropic has no separate reasoning-token field, so thinking
folds into `output_tokens` (hence the large count); the thinking text is carried
as a `reasoning` part inside `gen_ai.output.messages`, which the extractor does
not map.
