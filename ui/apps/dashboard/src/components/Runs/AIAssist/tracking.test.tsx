// @vitest-environment jsdom
import { cleanup, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useRunAITracking } from './tracking';

const { track } = vi.hoisted(() => ({ track: vi.fn() }));
vi.mock('@/utils/segment', () => ({ analytics: { track } }));
vi.mock('@clerk/tanstack-react-start', () => ({
  useOrganization: () => ({
    organization: { publicMetadata: { accountID: 'account-1' } },
  }),
}));
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe('Runs AI tracking', () => {
  it.each(['inline', 'standalone'] as const)(
    'identifies debugging engagement in %s details without run context',
    (view) => {
      const { result } = renderHook(() => useRunAITracking('debug', view));
      result.current({ action: 'prompt_copied' });
      expect(track.mock.lastCall?.[1]).toEqual({
        feature: 'runs-ai-assist',
        account_id: 'account-1',
        action: 'prompt_copied',
        surface: 'run_details',
        template_version: 'run_debug_v1',
        detail_view: view,
      });
    },
  );

  it('routes discovery through Segment with an explicit account ID', () => {
    const { result, rerender } = renderHook(() => useRunAITracking('fn'));
    result.current({ action: 'opened' });
    rerender();
    result.current({ action: 'prompt_copied' });
    expect(track).toHaveBeenLastCalledWith('Assistant Action Performed', {
      feature: 'runs-ai-assist',
      account_id: 'account-1',
      action: 'prompt_copied',
      surface: 'function_runs',
      template_version: 'basic_v1',
    });
  });

  it('tracks opening the ChatGPT listing as a setup handoff', () => {
    const { result } = renderHook(() => useRunAITracking('debug', 'inline'));
    result.current({ action: 'setup_opened', client: 'chatgpt' });
    expect(track).toHaveBeenCalledWith('Assistant Action Performed', {
      feature: 'runs-ai-assist',
      account_id: 'account-1',
      action: 'setup_opened',
      client: 'chatgpt',
      surface: 'run_details',
      template_version: 'run_debug_v1',
      detail_view: 'inline',
    });
  });

  it('handles asynchronous transport failures', async () => {
    const { result } = renderHook(() => useRunAITracking('env'));
    track.mockRejectedValueOnce(new Error('offline'));
    result.current({ action: 'opened' });
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(track).toHaveBeenCalledWith('Assistant Action Performed', {
      feature: 'runs-ai-assist',
      account_id: 'account-1',
      action: 'opened',
      surface: 'environment_runs',
      template_version: 'basic_v1',
    });
  });

  it('does not interrupt actions on tracking failure', () => {
    const { result } = renderHook(() => useRunAITracking('env'));
    track.mockImplementationOnce(() => {
      throw new Error('offline');
    });
    expect(() =>
      result.current({ action: 'setup_clicked', client: 'codex_cli' }),
    ).not.toThrow();
  });
});
