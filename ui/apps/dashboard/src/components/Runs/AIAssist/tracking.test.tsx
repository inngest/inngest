// @vitest-environment jsdom
import { cleanup, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useRunAITracking } from './tracking';

const { trackEvent } = vi.hoisted(() => ({ trackEvent: vi.fn() }));
vi.mock('@/utils/tracking', () => ({
  useTrackingUser: () => ({
    external_id: 'person-1',
    account_id: 'account-1',
    name: null,
  }),
  trackEvent,
}));
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  sessionStorage.clear();
});

describe('Runs AI tracking', () => {
  it('uses the existing identity and emits only categorical feature data', () => {
    const { result, rerender } = renderHook(() =>
      useRunAITracking('fn', '?search=private&filterApp=secret'),
    );
    result.current.track({ action: 'opened' });
    const visitID = result.current.getVisitID();
    rerender();
    result.current.track({ action: 'prompt_copied' });
    expect(trackEvent).toHaveBeenLastCalledWith({
      name: 'app/runs.ai_assist.action',
      v: '1',
      user: { external_id: 'person-1', account_id: 'account-1', name: null },
      data: {
        action: 'prompt_copied',
        surface: 'function_runs',
        template_version: 'basic_v1',
        visit_id: visitID,
      },
    });
    expect(JSON.stringify(trackEvent.mock.calls)).not.toContain('private');
    expect(JSON.stringify(trackEvent.mock.calls)).not.toContain('secret');
  });

  it('deduplicates exposure after setup returns to the same visit', () => {
    const search = '?aiVisit=12345678-1234-1234-1234-123456789abc';
    const first = renderHook(() => useRunAITracking('env', search));
    first.result.current.track({ action: 'viewed' });
    first.unmount();
    const second = renderHook(() => useRunAITracking('env', search));
    second.result.current.track({ action: 'viewed' });
    expect(trackEvent).toHaveBeenCalledTimes(1);
  });

  it('preserves the visit on setup and return and does not interrupt actions on tracking failure', () => {
    const visitID = '12345678-1234-1234-1234-123456789abc';
    const { result } = renderHook(() =>
      useRunAITracking('env', `?aiVisit=${visitID}`),
    );
    expect(result.current.getVisitID()).toBe(visitID);
    trackEvent.mockImplementationOnce(() => {
      throw new Error('offline');
    });
    expect(() =>
      result.current.track({ action: 'setup_clicked', client: 'codex_cli' }),
    ).not.toThrow();
  });
});
