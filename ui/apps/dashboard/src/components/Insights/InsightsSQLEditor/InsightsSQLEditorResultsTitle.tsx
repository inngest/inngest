'use client';

import { Pill } from '@inngest/components/Pill';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { useQuery } from 'urql';

import { GetAccountEntitlementsDocument } from '@/gql/graphql';

const ROW_LIMIT = 1000;

export function InsightsSQLEditorResultsTitle() {
  const [{ data: entitlementsData }] = useQuery({ query: GetAccountEntitlementsDocument });
  const historyWindow = entitlementsData?.account.entitlements.history.limit ?? 7;

  return (
    <div className="mr-1 flex items-center gap-2">
      <span className="uppercase">Results</span>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Pill appearance="outlined" kind="info">
              Row limit : {ROW_LIMIT}
            </Pill>
          </span>
        </TooltipTrigger>
        <TooltipContent className="p-2 text-xs" side="bottom" sideOffset={3}>
          Results are currently limited to at most {ROW_LIMIT} rows.
        </TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            <Pill appearance="outlined" kind="info">
              History limit : {historyWindow} days
            </Pill>
          </span>
        </TooltipTrigger>
        <TooltipContent className="p-2 text-xs" side="bottom" sideOffset={3}>
          Results are limited to the past {historyWindow} days based on your plan.
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
