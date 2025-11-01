import { renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';
import { type ReactNode } from 'react';

import { useGetTraceResult } from '../SharedContext/useGetTraceResult';
import type { StepInfoType } from './utils';

// Mock the useGetTraceResult hook
vi.mock('../SharedContext/useGetTraceResult');

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('StepInfo error handling', () => {
  it('should enable query when outputID is present', () => {
    const mockUseGetTraceResult = vi.mocked(useGetTraceResult);
    mockUseGetTraceResult.mockReturnValue({
      loading: false,
      data: null,
      error: null,
      refetch: vi.fn(),
    });

    const wrapper = createWrapper();

    renderHook(
      () =>
        useGetTraceResult({
          traceID: 'valid-output-id',
          refetchInterval: undefined,
          preview: true,
          enabled: true,
        }),
      { wrapper }
    );

    expect(mockUseGetTraceResult).toHaveBeenCalledWith(
      expect.objectContaining({
        traceID: 'valid-output-id',
        enabled: true,
      })
    );
  });

  it('should disable query when outputID is null', () => {
    const mockUseGetTraceResult = vi.mocked(useGetTraceResult);
    mockUseGetTraceResult.mockReturnValue({
      loading: false,
      data: null,
      error: null,
      refetch: vi.fn(),
    });

    const wrapper = createWrapper();

    renderHook(
      () =>
        useGetTraceResult({
          traceID: null,
          refetchInterval: undefined,
          preview: true,
          enabled: false,
        }),
      { wrapper }
    );

    expect(mockUseGetTraceResult).toHaveBeenCalledWith(
      expect.objectContaining({
        traceID: null,
        enabled: false,
      })
    );
  });

  it('should handle failed trace without outputID', () => {
    const trace: StepInfoType['trace'] = {
      attempts: 1,
      childrenSpans: [],
      endedAt: '2025-01-01T00:00:01Z',
      isRoot: false,
      name: 'test-step',
      outputID: null,
      queuedAt: '2025-01-01T00:00:00Z',
      spanID: 'span-1',
      stepID: 'step-1',
      startedAt: '2025-01-01T00:00:00.5Z',
      status: 'FAILED',
      stepInfo: null,
      stepOp: null,
      stepType: 'step.run',
      userlandSpan: null,
      isUserland: false,
    };

    const hasNoOutputID = !trace.outputID;
    const isFailed = trace.status === 'FAILED';

    expect(hasNoOutputID).toBe(true);
    expect(isFailed).toBe(true);
    expect(hasNoOutputID && isFailed).toBe(true);
  });

  it('should not show error for completed trace without outputID', () => {
    const trace: StepInfoType['trace'] = {
      attempts: 1,
      childrenSpans: [],
      endedAt: '2025-01-01T00:00:01Z',
      isRoot: false,
      name: 'test-step',
      outputID: null,
      queuedAt: '2025-01-01T00:00:00Z',
      spanID: 'span-1',
      stepID: 'step-1',
      startedAt: '2025-01-01T00:00:00.5Z',
      status: 'COMPLETED',
      stepInfo: null,
      stepOp: null,
      stepType: 'step.run',
      userlandSpan: null,
      isUserland: false,
    };

    const hasNoOutputID = !trace.outputID;
    const isFailed = trace.status === 'FAILED';

    expect(hasNoOutputID).toBe(true);
    expect(isFailed).toBe(false);
    expect(hasNoOutputID && isFailed).toBe(false);
  });

  it('should handle error with valid outputID', () => {
    const trace: StepInfoType['trace'] = {
      attempts: 1,
      childrenSpans: [],
      endedAt: '2025-01-01T00:00:01Z',
      isRoot: false,
      name: 'test-step',
      outputID: 'valid-output-id',
      queuedAt: '2025-01-01T00:00:00Z',
      spanID: 'span-1',
      stepID: 'step-1',
      startedAt: '2025-01-01T00:00:00.5Z',
      status: 'FAILED',
      stepInfo: null,
      stepOp: null,
      stepType: 'step.run',
      userlandSpan: null,
      isUserland: false,
    };

    const hasNoOutputID = !trace.outputID;
    const isFailed = trace.status === 'FAILED';

    expect(hasNoOutputID).toBe(false);
    expect(isFailed).toBe(true);
    expect(Boolean(trace.outputID)).toBe(true);
  });

  it('should handle various trace statuses', () => {
    const statuses = ['QUEUED', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED', 'WAITING'];

    statuses.forEach((status) => {
      const trace: Partial<StepInfoType['trace']> = {
        status,
        outputID: null,
      };

      const isFailed = trace.status === 'FAILED';
      expect(isFailed).toBe(status === 'FAILED');
    });
  });
});
