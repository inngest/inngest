import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import { TooltipProvider } from '../Tooltip/Tooltip';
import { MetadataAttrs } from './MetadataAttrs';
import type { SpanMetadata } from './types';

const renderAttrs = (metadata: SpanMetadata[]) =>
  render(
    <TooltipProvider>
      <MetadataAttrs metadata={metadata} />
    </TooltipProvider>
  );

afterEach(() => {
  cleanup();
});

const aiMetadata: SpanMetadata = {
  scope: 'step_attempt',
  kind: 'inngest.ai',
  updatedAt: '2026-01-01T00:00:02Z',
  values: { request_model: 'claude-opus-4', input_tokens: 10, output_tokens: 20 },
};

const aiSummaryMetadata: SpanMetadata = {
  scope: 'run',
  kind: 'inngest.ai.summary',
  updatedAt: '2026-01-01T00:00:05Z',
  values: {
    input_tokens: 1200,
    output_tokens: 340,
    total_tokens: 1540,
    estimated_cost: 0.0234,
    models: ['claude-opus-4', 'gpt-4o'],
    partial: false,
  },
};

const httpTimingMetadata: SpanMetadata = {
  scope: 'step_attempt',
  kind: 'inngest.http.timing',
  updatedAt: '2026-01-01T00:00:02Z',
  values: {
    dns_lookup_ms: 1,
    tcp_connection_ms: 2,
    tls_handshake_ms: 3,
    server_processing_ms: 4,
    content_transfer_ms: 5,
    total_ms: 15,
  },
};

const httpMetadata: SpanMetadata = {
  scope: 'extended_trace',
  kind: 'inngest.http',
  updatedAt: '2026-01-01T00:00:02Z',
  values: { method: 'POST', domain: 'example.com', path: '/api' },
};

describe('MetadataAttrs', () => {
  it('gives multi-segment kinds their own heading instead of collapsing onto the parent kind', () => {
    renderAttrs([aiMetadata, aiSummaryMetadata, httpMetadata, httpTimingMetadata]);

    expect(screen.getByText('AI Metadata')).toBeTruthy();
    expect(screen.getByText('AI Summary')).toBeTruthy();
    expect(screen.getByText('HTTP Metadata')).toBeTruthy();
    expect(screen.getByText('HTTP Timing')).toBeTruthy();
  });

  it('renders every inngest.ai entry when several share the same scope and kind', () => {
    const secondAiMetadata: SpanMetadata = {
      ...aiMetadata,
      updatedAt: '2026-01-01T00:00:03Z',
      values: { request_model: 'gpt-4o', input_tokens: 55, output_tokens: 66 },
    };

    renderAttrs([aiMetadata, secondAiMetadata]);

    expect(screen.getAllByText('AI Metadata')).toHaveLength(2);
    expect(screen.getByText('claude-opus-4')).toBeTruthy();
    expect(screen.getByText('gpt-4o')).toBeTruthy();
  });

  it('renders the run-level AI summary values', () => {
    renderAttrs([aiSummaryMetadata]);

    expect(screen.getByText('total_tokens')).toBeTruthy();
    expect(screen.getByText('1540')).toBeTruthy();
    // Arrays go through the generic JSON.stringify path, as they do for
    // inngest.ai's finish_reasons.
    expect(screen.getByText(/claude-opus-4/)).toBeTruthy();
  });

  it('marks a partial AI summary as incomplete, and a complete one not', () => {
    renderAttrs([
      aiSummaryMetadata,
      {
        ...aiSummaryMetadata,
        values: { ...aiSummaryMetadata.values, partial: true },
      },
    ]);

    expect(screen.getAllByText('AI Summary')).toHaveLength(2);
    expect(screen.getAllByText(/partial — may exclude invoked child run usage/)).toHaveLength(1);
  });

  it('falls back to the kind suffix for kinds without a label', () => {
    renderAttrs([{ ...aiMetadata, kind: 'inngest.timing' as SpanMetadata['kind'] }]);

    expect(screen.getByText('Metadata (timing)')).toBeTruthy();
  });
});
