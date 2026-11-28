import { StrictMode, useEffect } from 'react';
import { act, cleanup, render, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { useQuickstartTracking } from './useQuickstartTracking';

const trackEvent = vi.hoisted(() => vi.fn());
vi.mock('@/hooks/useTracking', () => ({ useTracking: () => ({ trackEvent }) }));
beforeEach(() => {
  sessionStorage.clear();
  trackEvent.mockClear();
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function Milestone({ server = 'http://localhost:8288' }) {
  const track = useQuickstartTracking(server);
  useEffect(() => {
    track('app_ready', {}, true);
  }, [track]);
  return null;
}
it('deduplicates milestones across StrictMode, polling renders, and remounts', () => {
  const view = render(
    <StrictMode>
      <Milestone />
    </StrictMode>,
  );
  view.rerender(
    <StrictMode>
      <Milestone />
    </StrictMode>,
  );
  view.unmount();
  render(<Milestone />);
  expect(trackEvent).toHaveBeenCalledTimes(1);
  expect(trackEvent).toHaveBeenCalledWith('cli/onboarding.action', {
    flow: 'quickstart',
    version: 1,
    action: 'app_ready',
  });
});
it('counts intentional repeated interactions and sends categorical metadata only', () => {
  const { result } = renderHook(() =>
    useQuickstartTracking('http://localhost:8288'),
  );
  act(() => {
    result.current('manual_install_copy_clicked', {
      language: 'Go',
      package_manager: 'go',
    });
    result.current('manual_install_copy_clicked', {
      language: 'Go',
      package_manager: 'go',
    });
  });
  expect(trackEvent).toHaveBeenCalledTimes(2);
  expect(trackEvent).toHaveBeenCalledWith('cli/onboarding.action', {
    flow: 'quickstart',
    version: 1,
    action: 'manual_install_copy_clicked',
    language: 'Go',
    package_manager: 'go',
  });
});
it('waits for a resolved server and keeps different servers separate', () => {
  const view = render(<Milestone server="" />);
  expect(trackEvent).not.toHaveBeenCalled();
  view.rerender(<Milestone />);
  view.rerender(<Milestone server="http://localhost:9000" />);
  expect(trackEvent).toHaveBeenCalledTimes(2);
});
it('still works and deduplicates on the current mount when storage is unavailable', () => {
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
    throw new Error('blocked');
  });
  render(
    <StrictMode>
      <Milestone />
    </StrictMode>,
  );
  expect(trackEvent).toHaveBeenCalledTimes(1);
});
