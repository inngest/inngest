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
      <div className=" flex items-center justify-between px-4 py-4">
        <div className="text-basis text-sm font-medium">
          {selectedCell.columnName}
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
