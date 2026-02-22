import { useMemo } from 'react';
import { CodeBlock } from '@inngest/components/CodeBlock';
import { RiArrowDownSLine, RiArrowUpSLine } from '@remixicon/react';

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
      <div className=" flex items-center justify-between px-4 py-4">
        <div className="text-basis text-sm font-medium">
          {selectedCell.columnId}
        </div>
        <div className=" text-muted rounded px-1.5 py-0.5 text-xs font-medium uppercase">
          {selectedCell.columnType}
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-auto px-4 py-1">
        <CellValueCodeBlock
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

function CellValueCodeBlock({
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

    if (columnType === 'date') {
      const date = new Date(value);
      return {
        content: `${date.toLocaleString()}\n${date.toISOString()}`,
        language: 'plaintext',
      };
    }

    if (columnType === 'string') {
      const formatted = getFormattedJSONObjectOrArrayString(String(value));
      if (formatted !== null) {
        return { content: formatted, language: 'json' };
      }
      return { content: String(value), language: 'plaintext' };
    }

    return { content: String(value), language: 'plaintext' };
  }, [columnType, value]);

  return (
    <CodeBlock.Wrapper>
      <CodeBlock
        tab={{
          content,
          language,
          readOnly: true,
        }}
      />
    </CodeBlock.Wrapper>
  );
}
