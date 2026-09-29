// @vitest-environment jsdom

import { fireEvent, render, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import { InsightsColumnRole } from '@/gql/graphql';
import { useColumns } from './useColumns';

vi.mock('@/utils/usePathCreator', () => ({
  usePathCreator: () => ({
    runPopout: ({ runID }: { runID: string }) => `/env/test/runs/${runID}`,
    eventPopout: ({ eventID }: { eventID: string }) =>
      `/env/test/events/${eventID}`,
  }),
}));

const RUN_ID = 'run-id-from-role-metadata';
const EVENT_ID = 'event-id-from-role-metadata';

function renderCell(
  role: InsightsColumnRole,
  value: null | string,
  onClick?: () => void,
) {
  const data: InsightsFetchResult = {
    columns: [{ name: 'id', type: 'string', role }],
    rows: [{ id: 'row-0', values: { id: value } }],
    diagnostics: [],
  };
  const { result } = renderHook(() => useColumns(data));
  const cell = result.current.columns[0]?.cell;
  if (typeof cell !== 'function') throw new Error('Expected a cell renderer');

  return render(
    <div onClick={onClick}>{cell({ getValue: () => value } as never)}</div>,
  );
}

describe('useColumns', () => {
  it.each([
    {
      role: InsightsColumnRole.RunId,
      value: RUN_ID,
      href: `/env/test/runs/${RUN_ID}`,
    },
    {
      role: InsightsColumnRole.EventId,
      value: EVENT_ID,
      href: `/env/test/events/${EVENT_ID}`,
    },
  ])(
    'links a $role value to its detail page based on its role',
    ({ role, value, href }) => {
      const cellClick = vi.fn();
      const { container } = renderCell(role, value, cellClick);

      const link = container.querySelector('a');
      expect(link?.textContent).toBe(value);
      expect(link?.getAttribute('href')).toBe(href);

      fireEvent.click(link!);
      expect(cellClick).not.toHaveBeenCalled();
    },
  );

  it.each([
    {
      name: 'an unspecified role',
      role: InsightsColumnRole.Unspecified,
      value: RUN_ID,
    },
  ])('renders $name as plain text', ({ role, value }) => {
    const { container } = renderCell(role, value);

    expect(container.querySelector('a')).toBeNull();
    expect(container.textContent).toBe(value);
  });
});
