import { Disclosure } from '@headlessui/react';
import { Button } from '@inngest/components/Button';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { cn } from '@inngest/components/utils/classNames';
import { type ToolPartFor } from '@inngest/use-agent';
import { RiCheckLine, RiCloseLine, RiPlayLine } from '@remixicon/react';

import { useSQLEditorActions } from '@/components/Insights/InsightsSQLEditor/SQLEditorContext';
import { formatSQL } from '@/components/Insights/InsightsSQLEditor/utils';
import type { InsightsAgentConfig } from '../useInsightsAgent';

type GenerateSqlPart = ToolPartFor<InsightsAgentConfig, 'generate_sql'>;

function GenerateSqlToolUI({ part }: { part: GenerateSqlPart }) {
  const editorActions = useSQLEditorActions();

  const data = part.output?.data;
  const title = data?.title;
  const sql = data?.sql;
  // TODO: (ted): update ToolPartFor to include proper errorMessage type; casting here for now...
  const errorMessage = part.error ? (part.error as Error).message : null;

  if (data === undefined || sql === undefined) {
    return null;
  }

  // Format SQL for display
  const formattedSql = sql ? formatSQL(sql) : null;

  return (
    <div className="text-basis border-muted rounded-lg border bg-transparent px-3 py-2 text-sm">
      <Disclosure defaultOpen>
        <>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Disclosure.Button className="flex items-center justify-center">
                <div
                  className={cn(
                    'flex h-4 w-4 items-center justify-center rounded-full',
                    errorMessage ? 'bg-error' : 'bg-btnPrimary',
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

            {!!formattedSql && editorActions && (
              <div className="flex items-center gap-2">
                <OptionalTooltip tooltip="Run this query" side="bottom">
                  <Button
                    icon={
                      <RiPlayLine className="text-subtle size-8 scale-110" />
                    }
                    appearance="ghost"
                    size="small"
                    onClick={() => {
                      editorActions.setQueryAndRun(formattedSql);
                    }}
                  />
                </OptionalTooltip>
              </div>
            )}
          </div>
          <Disclosure.Panel className="mt-2">
            <pre className="bg-canvasSubtle mt-1 overflow-auto rounded p-2 text-xs">
              {formattedSql || errorMessage}
            </pre>
          </Disclosure.Panel>
        </>
      </Disclosure>
    </div>
  );
}

export const ToolMessage = ({ part }: { part: GenerateSqlPart }) => {
  return <GenerateSqlToolUI part={part} />;
};
