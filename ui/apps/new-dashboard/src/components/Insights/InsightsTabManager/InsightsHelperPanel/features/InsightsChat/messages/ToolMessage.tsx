"use client";

import { Disclosure } from "@headlessui/react";
import { Button } from "@inngest/components/Button";
import { OptionalTooltip } from "@inngest/components/Tooltip/OptionalTooltip";
import { cn } from "@inngest/components/utils/classNames";
import { type ToolPartFor } from "@inngest/use-agent";
import { RiCheckLine, RiCloseLine, RiPlayLine } from "@remixicon/react";

import type { InsightsAgentConfig } from "../useInsightsAgent";

type GenerateSqlPart = ToolPartFor<InsightsAgentConfig, "generate_sql">;

function GenerateSqlToolUI({
  part,
  onSqlChange,
  runQuery,
}: {
  part: GenerateSqlPart;
  onSqlChange: (sql: string) => void;
  runQuery: () => void;
}) {
  const data = part.output?.data;
  const title = data?.title;
  const sql = data?.sql;
  // TODO: (ted): update ToolPartFor to include proper errorMessage type; casting here for now...
  const errorMessage = part.error ? (part.error as Error).message : null;

  if (data === undefined || sql === undefined) {
    return null;
  }

  return (
    <div className="text-basis border-muted rounded-lg border bg-transparent px-3 py-2 text-sm">
      <Disclosure>
        <>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Disclosure.Button className="flex items-center justify-center">
                <div
                  className={cn(
                    "flex h-4 w-4 items-center justify-center rounded-full",
                    errorMessage ? "bg-error" : "bg-btnPrimary",
                  )}
                >
                  {errorMessage ? (
                    <RiCloseLine className="text-onContrast size-3" />
                  ) : (
                    <RiCheckLine className="text-onContrast size-3" />
                  )}
                </div>
              </Disclosure.Button>
              <span className="font-sm">{title || "Generated SQL"}</span>
            </div>

            {!!sql && (
              <div className="flex items-center gap-2">
                <OptionalTooltip tooltip="Run this query" side="bottom">
                  <Button
                    icon={
                      <RiPlayLine className="text-subtle size-8 scale-110" />
                    }
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
      </Disclosure>
    </div>
  );
}

export const ToolMessage = ({
  part,
  onSqlChange,
  runQuery,
}: {
  part: GenerateSqlPart;
  onSqlChange: (sql: string) => void;
  runQuery: () => void;
}) => {
  return (
    <GenerateSqlToolUI
      part={part}
      onSqlChange={onSqlChange}
      runQuery={runQuery}
    />
  );
};
