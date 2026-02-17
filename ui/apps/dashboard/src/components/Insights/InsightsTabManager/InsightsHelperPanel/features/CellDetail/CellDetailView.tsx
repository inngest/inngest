import { useMemo } from 'react';
import { CodeBlock } from '@inngest/components/CodeBlock';

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
      <div className="border-subtle border-b px-4 py-3">
        <div className="text-muted text-xs font-medium uppercase tracking-wide">
          {selectedCell.columnName}
        </div>
        <div className="text-subtle mt-0.5 text-xs">
          {selectedCell.columnType}
        </div>
      </div>
      <div className="min-h-0 flex-1">
        <CellValueCodeBlock
          columnType={selectedCell.columnType}
          value={selectedCell.value}
        />
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
    <CodeBlock
      tab={{
        content,
        language,
        readOnly: true,
      }}
      alwaysFullHeight
    />
  );
}
