import { useMemo } from 'react';
import { NewCodeBlock } from '@inngest/components/NewCodeBlock/NewCodeBlock';
import {
  usePathCreator,
  type PathCreator,
} from '@inngest/components/SharedContext/usePathCreator';
import {
  format,
  formatInTimeZone,
  isValidDate,
} from '@inngest/components/utils/date';
import {
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiFileCopyLine,
} from '@remixicon/react';
import { toast } from 'sonner';

import { InsightsColumnType } from '@/store/generated';
import { getFormattedJSONObjectOrArrayString } from './json';
import { buildPathHintMonacoLinks, type PathHintLike } from './pathHints';

export type CellDetailData = {
  rowIndex: number;
  columnId: string;
  columnType: InsightsColumnType;
  columnPathHints: readonly PathHintLike[] | null | undefined;
  value: unknown;
};

// Mirrors the dashboard Insights feature's
// InsightsTabManager/InsightsHelperPanel/features/CellDetail/CellDetailView.tsx,
// adapted to this backend's InsightsColumnType enum and raw (non-normalized)
// values. Rendered as HelperPanelFrame's content (InsightsPage.tsx) -- the
// frame itself owns the title bar and close button, so this only needs its
// own secondary header (column name + type), not a close control of its own.
export function CellDetail({
  selectedCell,
}: {
  selectedCell: CellDetailData | null;
}) {
  const { pathCreator } = usePathCreator();

  if (!selectedCell) {
    return (
      <div className="text-muted flex h-full items-center justify-center text-sm">
        Click a cell to view its contents
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex items-center justify-between px-4 py-4">
        <div className="text-basis text-sm font-medium">
          {selectedCell.columnId}
        </div>
        <div className="text-muted rounded px-1.5 py-0.5 text-xs font-medium uppercase">
          {selectedCell.columnType}
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-4 py-1">
        <CellValueDisplay
          columnType={selectedCell.columnType}
          columnPathHints={selectedCell.columnPathHints}
          value={selectedCell.value}
          pathCreator={pathCreator}
        />
      </div>
      <div className="border-subtle flex items-center gap-1.5 border-t px-4 py-3">
        <div className="flex items-center gap-0.5">
          <span className="border-subtle bg-canvasSubtle inline-flex h-5 w-5 items-center justify-center rounded border">
            <RiArrowDownSLine className="text-muted h-3.5 w-3.5" />
          </span>
          <span className="border-subtle bg-canvasSubtle inline-flex h-5 w-5 items-center justify-center rounded border">
            <RiArrowUpSLine className="text-muted h-3.5 w-3.5" />
          </span>
        </div>
        <span className="text-subtle text-xs">
          Use arrow keys to navigate the table.
        </span>
      </div>
    </div>
  );
}

function CellValueDisplay({
  columnType,
  columnPathHints,
  value,
  pathCreator,
}: {
  columnType: InsightsColumnType;
  columnPathHints: readonly PathHintLike[] | null | undefined;
  value: unknown;
  pathCreator: PathCreator;
}) {
  const { content, language } = useMemo(() => {
    if (value == null) {
      return { content: 'null', language: 'plaintext' };
    }

    if (columnType === InsightsColumnType.Json) {
      // JSON columns come through as either a raw JSON string or an
      // already-decoded object/array, depending on the driver -- handle
      // both rather than assuming one (String()-ing an object here would
      // print "[object Object]").
      if (typeof value !== 'string') {
        return { content: JSON.stringify(value, null, 2), language: 'json' };
      }
      const formatted = getFormattedJSONObjectOrArrayString(value);
      return { content: formatted ?? value, language: 'json' };
    }

    // dates are rendered by DateDisplay below instead.
    return { content: String(value), language: 'plaintext' };
  }, [columnType, value]);

  if (columnType === InsightsColumnType.Datetime && value != null) {
    return <DateDisplay value={value} />;
  }

  // A hinted value can be the column's own whole value (a bare scalar
  // cell, e.g. a plain-text run_id column, or a whole array --
  // pkg/duckdb/insights/tables.go's event_ids/sessions comments -- whose
  // elements the hint describes) or a value found at some JSON sub-path
  // of it (e.g. an OTel attribute key inside `attributes`, or a field of
  // each element inside `inputs`) -- buildPathHintMonacoLinks resolves
  // every pathHints entry the same way the backend itself does. Either
  // way, the text still renders exactly as it does today (plaintext or
  // JSON); only plain-click link overlays are added (NewCodeBlock's
  // monacoLinks).
  const monacoLinks = buildPathHintMonacoLinks(
    columnPathHints,
    value,
    columnType,
    pathCreator,
  );

  return (
    <div className="bg-codeEditor border-subtle h-full overflow-hidden rounded-lg border">
      <NewCodeBlock
        tab={{ content, language, readOnly: true }}
        scrollbarOptions={{ vertical: 'auto', horizontal: 'auto' }}
        monacoLinks={monacoLinks}
      />
    </div>
  );
}

function DateDisplay({ value }: { value: unknown }) {
  const date = value instanceof Date ? value : new Date(String(value));

  if (!isValidDate(date)) {
    return <span className="text-muted text-sm">Invalid date</span>;
  }

  const isoString = date.toISOString();
  const utcString = formatInTimeZone(date, 'UTC', 'dd MMM yyyy, HH:mm:ss');
  const localString = format(date, 'dd MMM yyyy, hh:mm:ss a');
  const unixMs = String(date.getTime());

  return (
    <div className="bg-canvasSubtle flex flex-col gap-3 rounded p-2 text-sm">
      <CopyableRow label="ISO 8601" value={isoString} />
      <CopyableRow label="UTC" value={utcString} />
      <CopyableRow label="LOCAL" value={localString} />
      <CopyableRow label="UNIX MS" value={unixMs} />
    </div>
  );
}

// Shared by DateDisplay's ISO/UTC/LOCAL/UNIX rows and HintedArrayItem's
// unlinked fallback (a session with no pathCreator.session, or a value
// that doesn't actually look like {key, id}).
function CopyableRow({ label, value }: { label: string; value: string }) {
  const copyToClipboard = () => {
    navigator.clipboard.writeText(value);
    toast.success('Copied to clipboard');
  };

  return (
    <div className="group flex flex-col rounded hover:bg-canvasMuted">
      <span className="text-muted">{label}</span>
      <div className="flex items-center gap-1">
        <span className="text-basis">{value}</span>
        <button
          onClick={copyToClipboard}
          className="text-muted hover:text-basis opacity-0 transition-opacity group-hover:opacity-100"
        >
          <RiFileCopyLine className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
