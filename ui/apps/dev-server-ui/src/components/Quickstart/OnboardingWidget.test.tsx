import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { OnboardingWidget } from './OnboardingWidget';
const fixture = vi.hoisted(() => ({
  state: {
    ready: true,
    isDevServer: true,
    completion: null as unknown,
    functions: [],
  },
}));
vi.mock('./QuickstartProvider', () => ({ useQuickstart: () => fixture.state }));
vi.mock('@tanstack/react-router', () => ({
  Link: ({ children }: any) => <a href="/?quickstart=true">{children}</a>,
}));
vi.mock('@inngest/components/Menu/MenuItem', () => ({
  MenuItem: ({ text, to }: any) => <a href={to}>{text}</a>,
}));
afterEach(() => {
  cleanup();
  fixture.state.completion = null;
});
it('offers quickstart throughout unfinished onboarding', () => {
  render(<OnboardingWidget collapsed={false} />);
  expect(
    screen.getByRole('link', { name: /Get started/ }).getAttribute('href'),
  ).toBe('/?quickstart=true');
});
it('removes the widget after verified completion', () => {
  fixture.state.completion = { runID: 'completed-run' };
  render(<OnboardingWidget collapsed={false} />);
  expect(screen.queryByRole('link')).toBeNull();
});
it('uses the existing collapsed menu entry with an explicit reopening URL', () => {
  render(<OnboardingWidget collapsed />);
  expect(
    screen.getByRole('link', { name: 'Open quickstart' }).getAttribute('href'),
  ).toBe('/?quickstart=true');
});
