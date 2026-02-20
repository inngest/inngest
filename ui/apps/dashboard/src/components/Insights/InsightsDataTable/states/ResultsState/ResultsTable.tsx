import {
  memo,
  useCallback,
  useEffect,
  useRef,
  type KeyboardEvent,
} from 'react';
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
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!selectedCell || !data) return;

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
          return;
      }

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
    },
    [selectedCell, data, openCellDetail, closeCellDetail],
  );

  // Scroll the selected cell into view after React commits the DOM update
  useEffect(() => {
    if (!selectedCell) return;
    containerRef.current
      ?.querySelector('td[data-selected="true"]')
      ?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, [selectedCell]);

  if (!assertData(data)) return null;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div
        ref={containerRef}
        tabIndex={0}
        onKeyDown={handleKeyDown}
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
