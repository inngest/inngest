/**
 * UserlandAttrs component tests.
 * Feature: exe-1233, Task: 004-userland-span-metadata
 *
 * Tests are written FIRST per TDD approach.
 */

import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import { UserlandAttrs } from './UserlandAttrs';
import type { UserlandSpanType } from './types';

afterEach(() => {
  cleanup();
});

function makeUserlandSpan(overrides: Partial<UserlandSpanType> = {}): UserlandSpanType {
  return {
    spanName: null,
    spanKind: null,
    serviceName: null,
    scopeName: null,
    scopeVersion: null,
    spanAttrs: null,
    resourceAttrs: null,
    ...overrides,
  };
}

describe('UserlandAttrs', () => {
  describe('span metadata header', () => {
    it('renders spanName when non-null with label "Span"', () => {
      const span = makeUserlandSpan({ spanName: 'HTTP GET /api/users' });
      render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.getByText('Span')).toBeTruthy();
      expect(screen.getByText('HTTP GET /api/users')).toBeTruthy();
    });

    it('renders spanKind as badge when non-null with label "Kind"', () => {
      const span = makeUserlandSpan({ spanKind: 'CLIENT' });
      render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.getByText('Kind')).toBeTruthy();
      expect(screen.getByText('CLIENT')).toBeTruthy();
    });

    it('renders serviceName when non-null with label "Service"', () => {
      const span = makeUserlandSpan({ serviceName: 'user-service' });
      render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.getByText('Service')).toBeTruthy();
      expect(screen.getByText('user-service')).toBeTruthy();
    });

    it('renders scopeName with label "Scope" when non-null', () => {
      const span = makeUserlandSpan({ scopeName: 'my-scope' });
      render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.getByText('Scope')).toBeTruthy();
      expect(screen.getByText('my-scope')).toBeTruthy();
    });

    it('renders scopeVersion with label "Version" when non-null', () => {
      const span = makeUserlandSpan({ scopeVersion: '1.0.0' });
      render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.getByText('Version')).toBeTruthy();
      expect(screen.getByText('1.0.0')).toBeTruthy();
    });

    it('renders both scope fields separately when both present', () => {
      const span = makeUserlandSpan({ scopeName: 'my-scope', scopeVersion: '1.0.0' });
      render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.getByText('Scope')).toBeTruthy();
      expect(screen.getByText('my-scope')).toBeTruthy();
      expect(screen.getByText('Version')).toBeTruthy();
      expect(screen.getByText('1.0.0')).toBeTruthy();
    });
  });

  describe('resource attributes', () => {
    it('renders resourceAttrs as key-value table when non-null', () => {
      const span = makeUserlandSpan({
        resourceAttrs: '{"host.name":"server-1","service.version":"2.0"}',
      });
      render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.getByText('host.name')).toBeTruthy();
      expect(screen.getByText('server-1')).toBeTruthy();
      expect(screen.getByText('service.version')).toBeTruthy();
      expect(screen.getByText('2.0')).toBeTruthy();
    });

    it('filters internal prefixes from resourceAttrs', () => {
      const span = makeUserlandSpan({
        resourceAttrs: '{"sys.internal":"hidden","host.name":"visible"}',
      });
      render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.queryByText('sys.internal')).toBeNull();
      expect(screen.getByText('host.name')).toBeTruthy();
      expect(screen.getByText('visible')).toBeTruthy();
    });

    it('handles malformed resourceAttrs JSON gracefully', () => {
      const span = makeUserlandSpan({ resourceAttrs: 'not-valid-json' });
      const { container } = render(<UserlandAttrs userlandSpan={span} />);
      expect(screen.queryByTestId('userland-resource-attrs-section')).toBeNull();
      // Component should render nothing since no other data is present
      expect(container.innerHTML).toBe('');
    });
  });

  describe('null handling', () => {
    it('hides all metadata labels when fields are null but renders spanAttrs', () => {
      const span = makeUserlandSpan({
        spanAttrs: '{"custom.key":"value"}',
      });
      render(<UserlandAttrs userlandSpan={span} />);
      // Metadata labels should not be present
      expect(screen.queryByText('Span')).toBeNull();
      expect(screen.queryByText('Kind')).toBeNull();
      expect(screen.queryByText('Service')).toBeNull();
      expect(screen.queryByText('Scope')).toBeNull();
      expect(screen.queryByText('Version')).toBeNull();
      // spanAttrs table should still render
      expect(screen.getByText('custom.key')).toBeTruthy();
      expect(screen.getByText('value')).toBeTruthy();
    });

    it('returns null when all fields are null', () => {
      const span = makeUserlandSpan();
      const { container } = render(<UserlandAttrs userlandSpan={span} />);
      expect(container.innerHTML).toBe('');
    });
  });
});
