'use client';

import { memo } from 'react';
import { cn } from '@inngest/components/utils/classNames';

type FormattedDataCellProps = {
  value: string;
  type: 'json' | 'number' | 'string';
};

function FormattedDataCellComponent({ value, type }: FormattedDataCellProps) {
  const prettyJson = type === 'json' ? safePrettyJSONObjectOrArray(value) : null;
  const needsMinWidth = type === 'json' || isLikelyWrapping(value);
  const isNumericValue = type === 'number';

  return (
    <div
      className={cn(
        'text-basis text-left text-sm font-medium',
        'w-fit overflow-x-hidden py-0 pr-[2px]',
        isNumericValue ? 'max-w-none' : 'max-w-[400px]',
        needsMinWidth ? 'min-w-[320px]' : '',
        'max-h-[150px] overflow-y-auto [scrollbar-width:thin] [&::-webkit-scrollbar]:w-1',
        isNumericValue
          ? 'whitespace-nowrap [overflow-wrap:normal]'
          : 'whitespace-pre-wrap [overflow-wrap:anywhere]'
      )}
    >
      {prettyJson ? (
        <pre className="text-basis whitespace-pre-wrap font-mono text-sm font-medium">
          {prettyJson}
        </pre>
      ) : (
        value
      )}
    </div>
  );
}

export const FormattedDataCell = memo(FormattedDataCellComponent);

function safePrettyJSONObjectOrArray(text: string): string | null {
  try {
    const parsed = JSON.parse(text);
    if (parsed !== null && typeof parsed === 'object') {
      return JSON.stringify(parsed, null, 2);
    }
    return null;
  } catch {
    return null;
  }
}

const WRAP_TRIGGER_LENGTH = 40;

function isLikelyWrapping(text: string): boolean {
  if (text.length > WRAP_TRIGGER_LENGTH) return true;
  if (/\n/.test(text)) return true;
  return false;
}
