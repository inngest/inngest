import { Pill } from '@inngest/components/Pill';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip/Tooltip';

const ROW_LIMIT = 1000;

type InsightsSQLEditorResultsTitleProps = {
  historyWindow?: number;
};

export function InsightsSQLEditorResultsTitle({
  historyWindow,
}: InsightsSQLEditorResultsTitleProps) {
  const historyText = historyWindow
    ? `${historyWindow} days`
    : 'specified by plan';
  const tooltipText = historyWindow
    ? `Based on your plan, results are limited to the past ${historyWindow} days.`
    : 'Historical data availability is specified by your plan.';

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
              History limit : {historyText}
            </Pill>
          </span>
        </TooltipTrigger>
        <TooltipContent className="p-2 text-xs" side="bottom" sideOffset={3}>
          {tooltipText}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
