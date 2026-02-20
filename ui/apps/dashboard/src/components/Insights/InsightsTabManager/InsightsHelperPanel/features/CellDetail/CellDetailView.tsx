import { useMemo } from 'react';
import {
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiFileCopyLine,
} from '@remixicon/react';
import { toast } from 'sonner';
import { format, formatInTimeZone } from '@inngest/components/utils/date';
import { NewCodeBlock } from '@inngest/components/NewCodeBlock/NewCodeBlock';

import { useCellDetailContext } from '@/components/Insights/CellDetailContext';
import { getFormattedJSONObjectOrArrayString } from '@/components/Insights/InsightsDataTable/states/ResultsState/json';

export function CellDetailView() {
  const { selectedCell } = useCellDetailContext();

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
          value={selectedCell.value}
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
  value,
}: {
  columnType: string;
  value: string | number | Date | null;
}) {
  const { content, language } = useMemo(() => {
    if (value == null) {
      return { content: 'null', language: 'plaintext' };
    }

    if (columnType === 'string') {
      const formatted = getFormattedJSONObjectOrArrayString(String(value));
      if (formatted !== null) {
        return { content: formatted, language: 'json' };
      }
      return { content: String(value), language: 'plaintext' };
    }

    // dates are ignored since we use a differnet component
    return { content: String(value ?? 'null'), language: 'plaintext' };
  }, [columnType, value]);

  if (columnType === 'date') {
    return <DateDisplay value={value} />;
  }

  return (
    <div className="bg-codeEditor h-full overflow-hidden rounded-lg">
      <NewCodeBlock
        tab={{
          content,
          language,
          readOnly: true,
        }}
      />
    </div>
  );
}

function DateDisplay({ value }: { value: string | number | null | Date }) {
  const date = value instanceof Date ? value : new Date(value);

  if (!(date instanceof Date) || isNaN(date.getTime())) {
    return <span className="text-muted text-sm">Invalid date</span>;
  }

  const isoString = date.toISOString();
  const utcString = formatInTimeZone(date, 'UTC', 'dd MMM yyyy, HH:mm:ss');
  const localString = format(date, 'dd MMM yyyy, hh:mm:ss a');
  const unixMs = String(date.getTime());

  return (
    <div className="bg-canvasSubtle flex flex-col gap-3 rounded p-2 text-sm">
      <DateRow label="ISO 8601" value={isoString} />
      <DateRow label="UTC" value={utcString} />
      <DateRow label="LOCAL" value={localString} />
      <DateRow label="UNIX MS" value={unixMs} />
    </div>
  );
}

function DateRow({ label, value }: { label: string; value: string }) {
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
