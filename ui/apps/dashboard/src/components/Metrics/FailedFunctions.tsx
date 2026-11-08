import { Button } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';
import { RiArrowRightUpLine } from '@remixicon/react';
import { useNavigate } from '@tanstack/react-router';

import { useEnvironment } from '@/components/Environments/environment-context';
import type { FunctionStatusMetricsQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import type { EntityLookup } from './Dashboard';
import { FailedRate } from './FailedRate';
import { getLineChartOptions, mapEntityLines } from './utils';

// SQL carried by the "Open in Insights" deep link. Kept local so the chart
// doesn't depend on a matching entry inside the Insights templates module —
// renaming or removing a built-in template can't silently break this button.
const INSIGHTS_QUERY = `SELECT
    data.function_id AS function_id,
    COUNT(*) as failed_count
FROM
    events
WHERE
    name = 'inngest/function.failed'
    AND ts > toUnixTimestamp64Milli(subtractDays(now64(), 1))
GROUP BY
    function_id
ORDER BY
    failed_count DESC`;
const INSIGHTS_QUERY_NAME = 'Failed function runs (24h)';

export const FailedFunctions = ({
  workspace,
  functions,
}: {
  workspace?: FunctionStatusMetricsQuery['workspace'];
  functions: EntityLookup;
}) => {
  const env = useEnvironment();
  const navigate = useNavigate();

  const metrics =
    workspace &&
    mapEntityLines(
      workspace.completedByFunction.metrics.filter(
        ({ tagValue }) => tagValue === 'Failed',
      ),
      functions,
    );

  return (
    <div className="bg-canvasBase border-subtle overflowx-hidden relative flex h-[384px] w-full flex-col rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between gap-x-2">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Failed Functions{' '}
          <Info
            text="Total number of failed runs in your environment, app or function."
            action={
              <Link
                className="text-sm"
                href="https://www.inngest.com/docs/platform/monitor/observability-metrics#failed-functions"
                target="_new"
              >
                Learn more about Inngest functions.
              </Link>
            }
          />
        </div>
        <Button
          size="small"
          kind="secondary"
          appearance="outlined"
          icon={<RiArrowRightUpLine />}
          iconSide="left"
          label="Open in Insights"
          // Programmatic client-side navigation rather than a Link `to` with
          // an embedded query string. Passing `search` as an object lets
          // TanStack route it through `stringifySearch`, which preserves
          // newlines (`\n` -> `%0A`). Embedding `?sql=...` inside a string
          // `to` makes the path resolver treat the query as part of the
          // pathname, and @tanstack/history's sanitizePath then strips out
          // all ASCII control chars — including the SQL's newlines.
          onClick={() =>
            navigate({
              to: pathCreator.insights({ envSlug: env.slug }),
              search: {
                sql: INSIGHTS_QUERY,
                name: INSIGHTS_QUERY_NAME,
              },
            })
          }
        />
      </div>
      <div className="flex h-full flex-row items-center">
        <Chart
          option={metrics ? getLineChartOptions(metrics) : {}}
          className="h-[100%] w-full md:w-[75%]"
          group="metricsDashboard"
        />
        <FailedRate workspace={workspace} functions={functions} />
      </div>
    </div>
  );
};
