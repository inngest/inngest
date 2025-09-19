'use client';

import { Disclosure } from '@headlessui/react';
import { Button } from '@inngest/components/Button';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';
import type { ToolCallUIPart, ToolResultPayload } from '@inngest/use-agents';
import { RiCheckLine, RiCloseLine, RiPlayLine } from '@remixicon/react';

import type { GenerateSqlResult } from '@/app/api/inngest/functions/agents/types';

function GenerateSqlToolUI({
  part,
  onSqlChange,
  runQuery,
}: {
  part: ToolCallUIPart;
  onSqlChange: (sql: string) => void;
  runQuery: () => void;
}) {
  const getToolData = (toolPart: ToolCallUIPart): { title: string | null; sql: string | null } => {
    if (toolPart.state !== 'output-available') {
      return { title: null, sql: null };
    }
    const output = part.output as ToolResultPayload<GenerateSqlResult>;
    const title = output?.data.title;
    const sql = output?.data.sql;
    if (!title || !sql) {
      return { title: 'Error generating SQL', sql: null };
    }
    return {
      title: title.trim(),
      sql: sql.trim(),
    };
  };

  const { title, sql } = getToolData(part);
  const errorMessage = part.error ? (part.error as Error).message : null;

  return (
    <div className="text-text-basis border-muted rounded-md border bg-transparent px-3 py-2 text-sm">
      <Disclosure>
        {() => (
          <>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Disclosure.Button className="flex items-center justify-center">
                  <div
                    className={cn(
                      'flex h-4 w-4 items-center justify-center rounded-full',
                      errorMessage ? 'bg-error' : 'bg-[#65BD8B]'
                    )}
                  >
                    {errorMessage ? (
                      <RiCloseLine className="text-onContrast size-3" />
                    ) : (
                      <RiCheckLine className="text-onContrast size-3" />
                    )}
                  </div>
                </Disclosure.Button>
                <span className="font-sm">{title || 'Generated SQL'}</span>
              </div>

              {!!sql && (
                <div className="flex items-center gap-2">
                  {/* <Button
                    icon={<RiEdit2Line className="size-4" />}
                    appearance="ghost"
                    size="small"
                    onClick={() => onSqlChange(sql)}
                  /> */}
                  <OptionalTooltip tooltip="Run this query" side="bottom">
                    <Button
                      icon={<RiPlayLine className="text-muted size-8 scale-110" />}
                      appearance="ghost"
                      size="small"
                      onClick={() => {
                        onSqlChange(sql);
                        try {
                          runQuery();
                        } catch {}
                      }}
                    />
                  </OptionalTooltip>
                </div>
              )}
            </div>
            <Disclosure.Panel className="mt-2">
              <pre className="bg-canvasSubtle mt-1 overflow-auto rounded p-2 text-xs">
                {sql || errorMessage}
              </pre>
            </Disclosure.Panel>
          </>
        )}
      </Disclosure>
    </div>
  );
}

export const ToolMessage = ({
  part,
  onSqlChange,
  runQuery,
}: {
  part: ToolCallUIPart;
  onSqlChange: (sql: string) => void;
  runQuery: () => void;
}) => {
  if (part.toolName !== 'generate_sql') {
    return null;
  }

  return <GenerateSqlToolUI part={part} onSqlChange={onSqlChange} runQuery={runQuery} />;
};
