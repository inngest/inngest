import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { AIMetadataNudge, AISummaryAttrs, traceHasAIMetadata } from './AIMetadata';
import type { SpanMetadata, SpanMetadataInngestAISummary, Trace } from './types';

// jsdom lacks ResizeObserver, which Pill uses for truncation detection.
vi.stubGlobal(
  'ResizeObserver',
  class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
);

const booleanFlagMock = vi.hoisted(() => vi.fn());
const pathCreatorMock = vi.hoisted(() => vi.fn());

vi.mock('../SharedContext/useBooleanFlag', () => ({
  useBooleanFlag: () => ({ booleanFlag: booleanFlagMock }),
}));

vi.mock('../SharedContext/usePathCreator', () => ({
  usePathCreator: () => ({ pathCreator: pathCreatorMock() }),
}));

const renderNudge = ({
  aiOverviewEnabled,
  hasAIOverviewPath = true,
}: {
  aiOverviewEnabled: boolean;
  hasAIOverviewPath?: boolean;
}) => {
  booleanFlagMock.mockReturnValue({ isReady: true, value: aiOverviewEnabled });
  pathCreatorMock.mockReturnValue(
    hasAIOverviewPath ? { aiOverview: () => '/env/prod/ai-overview' } : {}
  );

  render(<AIMetadataNudge />);
};

afterEach(() => {
  cleanup();
});

const makeSpan = (overrides: Partial<Trace> = {}): Trace => ({
  attempts: 0,
  endedAt: '2026-01-01T00:00:05Z',
  isRoot: false,
  name: 'span',
  outputID: null,
  queuedAt: '2026-01-01T00:00:00Z',
  scheduledAt: null,
  spanID: 'span-1',
  startedAt: '2026-01-01T00:00:01Z',
  status: 'COMPLETED',
  stepInfo: null,
  userlandSpan: null,
  isUserland: false,
  ...overrides,
});

const aiMetadata: SpanMetadata = {
  scope: 'extended_trace',
  kind: 'inngest.ai',
  updatedAt: '2026-01-01T00:00:02Z',
  values: { request_model: 'claude-opus-4', input_tokens: 10, output_tokens: 20 },
};

const timingMetadata: SpanMetadata = {
  scope: 'step_attempt',
  kind: 'inngest.timing',
  updatedAt: '2026-01-01T00:00:02Z',
  values: { total_inngest_ms: 12 },
};

describe('traceHasAIMetadata', () => {
  it('finds AI metadata nested under descendant spans', () => {
    const trace = makeSpan({
      isRoot: true,
      metadata: [timingMetadata],
      childrenSpans: [makeSpan({ childrenSpans: [makeSpan({ metadata: [aiMetadata] })] })],
    });

    expect(traceHasAIMetadata(trace)).toBe(true);
  });

  it('reports no AI metadata when only other kinds are present', () => {
    const trace = makeSpan({
      isRoot: true,
      metadata: [timingMetadata],
      childrenSpans: [makeSpan({ metadata: [] })],
    });

    expect(traceHasAIMetadata(trace)).toBe(false);
  });
});

const aiSummaryMetadata: SpanMetadataInngestAISummary = {
  scope: 'run',
  kind: 'inngest.ai.summary',
  updatedAt: '2026-01-01T00:00:05Z',
  values: {
    input_tokens: 1200,
    output_tokens: 3400,
    total_tokens: 4600,
    estimated_cost: 0.004235,
    models: ['claude-opus-4', 'gpt-4o-mini'],
    providers: ['anthropic', 'openai'],
  },
};

const renderSummary = (metadata: SpanMetadataInngestAISummary) =>
  render(
    <TooltipProvider>
      <AISummaryAttrs metadata={metadata} />
    </TooltipProvider>
  );

describe('AISummaryAttrs', () => {
  it('renders formatted token totals, cost, models, and providers', () => {
    renderSummary(aiSummaryMetadata);

    expect(screen.getByText('AI Usage')).toBeTruthy();
    expect(screen.getByText('4,600')).toBeTruthy();
    expect(screen.getByText('1,200')).toBeTruthy();
    expect(screen.getByText('3,400')).toBeTruthy();
    expect(screen.getByText('$0.004235')).toBeTruthy();
    // Pill renders its text twice: once visibly, once in a hidden measuring span.
    expect(screen.getAllByText('claude-opus-4').length).toBeGreaterThan(0);
    expect(screen.getAllByText('gpt-4o-mini').length).toBeGreaterThan(0);
    expect(screen.getAllByText('anthropic').length).toBeGreaterThan(0);
    expect(screen.getAllByText('openai').length).toBeGreaterThan(0);
  });

  it('renders a zero-valued optional token count instead of omitting it', () => {
    renderSummary({
      ...aiSummaryMetadata,
      values: { ...aiSummaryMetadata.values, cache_read_tokens: 0 },
    });

    expect(screen.getByText('Cache read tokens')).toBeTruthy();
    expect(screen.getByText('0')).toBeTruthy();
  });

  it('omits rows for absent optional fields', () => {
    renderSummary({
      ...aiSummaryMetadata,
      values: { input_tokens: 1, output_tokens: 2, total_tokens: 3 },
    });

    expect(screen.getByText('Total tokens')).toBeTruthy();
    expect(screen.queryByText('Cache read tokens')).toBeNull();
    expect(screen.queryByText('Cache creation tokens')).toBeNull();
    expect(screen.queryByText('Reasoning tokens')).toBeNull();
    expect(screen.queryByText('Estimated cost')).toBeNull();
    expect(screen.queryByText('Models')).toBeNull();
    expect(screen.queryByText('Providers')).toBeNull();
  });
});

describe('AIMetadataNudge', () => {
  it('always offers the documentation links', () => {
    renderNudge({ aiOverviewEnabled: false });

    expect(screen.getByRole('link', { name: 'Set up AI metadata' }).getAttribute('href')).toContain(
      '/docs/examples/ai-metadata-quickstart'
    );
    expect(screen.getByRole('link', { name: 'Add custom metadata' })).toBeTruthy();
  });

  it('links to the AI Overview when the dashboard is enabled', () => {
    renderNudge({ aiOverviewEnabled: true });

    expect(screen.getByRole('link', { name: 'View AI Overview' }).getAttribute('href')).toBe(
      '/env/prod/ai-overview'
    );
  });

  it('omits the AI Overview link when the dashboard is disabled', () => {
    renderNudge({ aiOverviewEnabled: false });

    expect(screen.queryByRole('link', { name: 'View AI Overview' })).toBeNull();
  });

  it('omits the AI Overview link where the route does not exist', () => {
    renderNudge({ aiOverviewEnabled: true, hasAIOverviewPath: false });

    expect(screen.queryByRole('link', { name: 'View AI Overview' })).toBeNull();
  });
});
