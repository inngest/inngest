import { Fragment } from 'react';
import { Pill, PillContent } from '@inngest/components/Pill';
import type { PathCreator } from '@inngest/components/SharedContext/usePathCreator';
import { LinkCell, TextCell } from '@inngest/components/Table';
import { describe, expect, it } from 'vitest';

import { InsightsColumnHint, InsightsColumnType } from '@/store/generated';
import { renderCell } from './InsightsQueryTab';

const pathCreator: PathCreator = {
  app: ({ externalAppID }) => `/apps/${externalAppID}`,
  eventPopout: ({ eventID }) => `/event?eventID=${eventID}`,
  function: ({ functionSlug }) => `/functions/${functionSlug}`,
  runPopout: ({ runID }) => `/run?runID=${runID}`,
};

describe('renderCell', () => {
  it('renders a monospaced run link for a RUN_ID hint', () => {
    const element = renderCell(
      'run-123',
      InsightsColumnType.String,
      InsightsColumnHint.RunId,
      pathCreator,
    );

    expect(element.type).toBe(LinkCell);
    expect(element.props.href).toBe('/run?runID=run-123');
    expect(element.props.className).toBe('font-mono');
    expect(element.props.children).toBe('run-123');
  });

  it('renders a monospaced event link for an EVENT_ID hint', () => {
    const element = renderCell(
      'event-456',
      InsightsColumnType.String,
      InsightsColumnHint.EventId,
      pathCreator,
    );

    expect(element.type).toBe(LinkCell);
    expect(element.props.href).toBe('/event?eventID=event-456');
    expect(element.props.className).toBe('font-mono');
    expect(element.props.children).toBe('event-456');
  });

  it('falls back to type-based rendering when there is no hint', () => {
    const element = renderCell(
      'plain value',
      InsightsColumnType.String,
      null,
      pathCreator,
    );

    expect(element.type).toBe(TextCell);
    expect(element.props.children).toBe('plain value');
  });

  it('renders a clickable app badge for an APP_ID hint', () => {
    const element = renderCell(
      'my-app',
      InsightsColumnType.String,
      InsightsColumnHint.AppId,
      pathCreator,
    );

    expect(element.type).toBe(Pill);
    expect(element.props.href).toBe('/apps/my-app');
    const content = element.props.children;
    expect(content.type).toBe(PillContent);
    expect(content.props.type).toBe('APP');
    expect(content.props.children).toBe('my-app');
  });

  it('renders a clickable function badge for a FUNCTION_ID hint', () => {
    const element = renderCell(
      'my-fn',
      InsightsColumnType.String,
      InsightsColumnHint.FunctionId,
      pathCreator,
    );

    expect(element.type).toBe(Pill);
    expect(element.props.href).toBe('/functions/my-fn');
    const content = element.props.children;
    expect(content.type).toBe(PillContent);
    expect(content.props.type).toBe('FUNCTION');
    expect(content.props.children).toBe('my-fn');
  });

  it('renders NULL for nullish values regardless of hint', () => {
    const element = renderCell(
      null,
      InsightsColumnType.String,
      InsightsColumnHint.RunId,
      pathCreator,
    );

    expect(element.type).toBe(TextCell);
    expect(element.props.children).toBe('ᴺᵁᴸᴸ');
  });

  it('renders one comma-separated link with no ellipsis for a hinted array with exactly 1 item', () => {
    const ids = ['event-1'];
    const element = renderCell(
      ids,
      InsightsColumnType.Json,
      InsightsColumnHint.EventId,
      pathCreator,
    );

    expect(element.type).toBe(TextCell);
    expect(element.props.className).toBe('font-mono');
    const [fragments, hasMore] = element.props.children;
    expect(fragments).toHaveLength(1);
    fragments.forEach((fragment: any, i: number) => {
      expect(fragment.type).toBe(Fragment);
      const [comma, link] = fragment.props.children;
      expect(comma).toBe(i > 0 && ', ');
      expect(link.type).toBe(LinkCell);
      expect(link.props.href).toBe(`/event?eventID=${ids[i]}`);
      expect(link.props.children).toBe(ids[i]);
    });
    expect(hasMore).toBe(false);
  });

  it('renders only the first link plus a comma-separated ellipsis for a hinted array with more than 1 item', () => {
    const ids = ['run-1', 'run-2', 'run-3', 'run-4', 'run-5', 'run-6', 'run-7'];
    const element = renderCell(
      ids,
      InsightsColumnType.Json,
      InsightsColumnHint.RunId,
      pathCreator,
    );

    const [fragments, hasMore] = element.props.children;
    expect(fragments).toHaveLength(1);
    const [, link] = fragments[0].props.children;
    expect(link.type).toBe(LinkCell);
    expect(link.props.children).toBe('run-1');
    expect(hasMore).toBe(', …');
  });
});
