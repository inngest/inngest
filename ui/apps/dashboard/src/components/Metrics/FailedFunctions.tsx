import { Button } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';
import { RiArrowRightUpLine } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import type { FunctionStatusMetricsQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import type { EntityLookup } from './Dashboard';
import { FailedRate } from './FailedRate';
import { getLineChartOptions, mapEntityLines, sum } from './utils';
import type { FileRouteTypes } from '@tanstack/react-router';

export type CompletedType =
  FunctionStatusMetricsQuery['workspace']['completed'];
export type CompletedMetricsType =
  FunctionStatusMetricsQuery['workspace']['completed']['metrics'];

const filter = ({ metrics }: CompletedType) =>
  metrics.filter(({ tagValue }) => tagValue === 'Failed');

const sort = (metrics: CompletedMetricsType) =>
  metrics.sort(({ data: data1 }, { data: data2 }) => sum(data2) - sum(data1));

const mapFailed = (
  { completed }: FunctionStatusMetricsQuery['workspace'],
  entities: EntityLookup,
) => {
  const failed = sort(filter(completed));
  return mapEntityLines(failed, entities);
};

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
  entities,
  functions,
}: {
  workspace?: FunctionStatusMetricsQuery['workspace'];
  entities: EntityLookup;
  functions: EntityLookup;
}) => {
  const env = useEnvironment();

  const metrics = workspace && mapFailed(workspace, entities);

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
          to={
            `${pathCreator.insights({
              envSlug: env.slug,
            })}?sql=${encodeURIComponent(
              INSIGHTS_QUERY,
            )}&name=${encodeURIComponent(
              INSIGHTS_QUERY_NAME,
            )}` as FileRouteTypes['to']
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
