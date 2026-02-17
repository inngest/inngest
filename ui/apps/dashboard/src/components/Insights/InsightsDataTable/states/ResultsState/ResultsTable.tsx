import { memo, useCallback, useEffect, useRef } from 'react';
import { Table } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import {
  useCellDetailContext,
  type CellDetailData,
} from '@/components/Insights/CellDetailContext';
import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import { ResultsTableFooter, assertData } from './ResultsTableFooter';
import { useColumns } from './useColumns';

type InsightsEntry = InsightsFetchResult['rows'][number];
type InsightsColumnValue = InsightsEntry['values'][string];
type InsightsTableProps = {
  cellClassName: string;
  columns: ColumnDef<InsightsEntry, InsightsColumnValue>[];
  data: InsightsEntry[];
  onCellClick?: (rowIndex: number, columnId: string, value: unknown) => void;
  selectedCell?: CellDetailData | null;
};

function InsightsTable({
  cellClassName,
  columns,
  data,
  onCellClick,
  selectedCell,
}: InsightsTableProps) {
  return (
    <Table<InsightsEntry>
      cellClassName={cellClassName}
      columns={columns}
      data={data}
      enableColumnSizing
      selectedCell={selectedCell}
      onCellClick={onCellClick}
    />
  );
}

const MemoizedInsightsTable = memo(InsightsTable);

export function ResultsTable() {
  const containerRef = useRef<HTMLDivElement>(null);
  const { data } = useInsightsStateMachineContext();
  const { columns } = useColumns(data);
  const { openCellDetail, closeCellDetail, selectedCell } =
    useCellDetailContext();

  const handleCellClick = useCallback(
    (rowIndex: number, columnId: string, value: unknown) => {
      if (!data) return;
      const col = data.columns.find((c) => c.name === columnId);
      openCellDetail({
        rowIndex,
        columnId,
        columnType: col?.type ?? 'string',
        value: value as string | number | Date | null,
      });
    },
    [data, openCellDetail],
  );

  // Keyboard navigation: arrow keys move between cells, Escape deselects
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // Only handle keys when a cell is selected
      if (!selectedCell || !data) return;

      // Build an ordered list of column names so we can index into it
      const colNames = data.columns.map((c) => c.name);
      const colIndex = colNames.indexOf(selectedCell.columnId);
      if (colIndex === -1) return;

      let nextRow = selectedCell.rowIndex;
      let nextColIndex = colIndex;

      switch (e.key) {
        case 'ArrowUp':
          nextRow = Math.max(0, nextRow - 1);
          break;
        case 'ArrowDown':
          nextRow = Math.min(data.rows.length - 1, nextRow + 1);
          break;
        case 'ArrowLeft':
          nextColIndex = Math.max(0, nextColIndex - 1);
          break;
        case 'ArrowRight':
          nextColIndex = Math.min(colNames.length - 1, nextColIndex + 1);
          break;
        case 'Escape':
          closeCellDetail();
          return;
        default:
          return; // Ignore all other keys
      }

      // Prevent the scroll container from scrolling on arrow keys
      e.preventDefault();

      const nextColumnId = colNames[nextColIndex];
      const col = data.columns[nextColIndex];
      if (!nextColumnId || !col) return;

      const value = data.rows[nextRow]?.values[nextColumnId] ?? null;

      openCellDetail({
        rowIndex: nextRow,
        columnId: nextColumnId,
        columnType: col.type,
        value,
      });

      // Wait one frame for React to re-render, then scroll the new cell into view
      requestAnimationFrame(() => {
        container.querySelector('td[data-selected="true"]')?.scrollIntoView({
          block: 'nearest',
          inline: 'nearest',
        });
      });
    };

    container.addEventListener('keydown', handleKeyDown);
    return () => container.removeEventListener('keydown', handleKeyDown);
  }, [selectedCell, data, openCellDetail, closeCellDetail]);

  if (!assertData(data)) return null;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div
        ref={containerRef}
        tabIndex={0}
        className="flex-1 overflow-auto overscroll-none outline-none"
        id="insights-table-container"
      >
        <MemoizedInsightsTable
          cellClassName="[&:not(:first-child)]:border-l [&:not(:first-child)]:border-light box-border align-top px-4 py-2.5 overflow-hidden text-ellipsis whitespace-nowrap"
          columns={columns}
          data={data.rows}
          onCellClick={handleCellClick}
          selectedCell={selectedCell}
        />
      </div>
      <ResultsTableFooter />
    </div>
  );
}
