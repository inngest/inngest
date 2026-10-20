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
});

describe('Runs AI tracking', () => {
  it('uses the existing identity and emits only categorical feature data', () => {
    const { result, rerender } = renderHook(() => useRunAITracking('fn'));
    result.current({ action: 'opened' });
    rerender();
    result.current({ action: 'prompt_copied' });
    expect(trackEvent).toHaveBeenLastCalledWith({
      name: 'app/runs.ai_assist.action',
      v: '1',
      user: { external_id: 'person-1', account_id: 'account-1', name: null },
      data: {
        action: 'prompt_copied',
        surface: 'function_runs',
        template_version: 'basic_v1',
      },
    });
    expect(JSON.stringify(trackEvent.mock.calls)).not.toContain('private');
    expect(JSON.stringify(trackEvent.mock.calls)).not.toContain('secret');
  });

  it('does not interrupt actions on tracking failure', () => {
    const { result } = renderHook(() => useRunAITracking('env'));
    trackEvent.mockImplementationOnce(() => {
      throw new Error('offline');
    });
    expect(() =>
      result.current({ action: 'setup_clicked', client: 'codex_cli' }),
    ).not.toThrow();
  });
});
