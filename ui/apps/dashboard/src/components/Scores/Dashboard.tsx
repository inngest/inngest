import type { RangeChangeProps } from '@inngest/components/DatePicker/RangePicker';
import EntityFilter from '@inngest/components/Filter/EntityFilter';
import { TimeFilter } from '@inngest/components/Filter/TimeFilter';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Switch } from '@inngest/components/Switch';
import {
  useBatchedSearchParams,
  useSearchParam,
  useStringArraySearchParam,
} from '@inngest/components/hooks/useSearchParams';
import {
  durationToString,
  parseDuration,
  toDate,
} from '@inngest/components/utils/date';
import { useQuery } from 'urql';

import { graphql } from '@/gql';

export type ScoreMetric = {
  key: string;
  label: string;
};

// TODO: replace with backend-supplied list of score metric keys.
const PLACEHOLDER_METRICS: ScoreMetric[] = [
  { key: 'metric_1', label: 'metric_1' },
  { key: 'metric_2', label: 'metric_2' },
  { key: 'metric_3', label: 'metric_3' },
  { key: 'metric_4', label: 'metric_4' },
];

// Positional palette; index assigned at render time so colors aren't bound
// to any particular metric key.
const LEGEND_COLORS = [
  '#2389f1',
  '#6222df',
  '#ec9923',
  '#eab308',
  '#2c9b63',
  '#f54a3f',
];

const DEFAULT_DURATION = { hours: 24 };

const ScoresLookupDocument = graphql(`
  query ScoresLookup($envSlug: String!, $page: Int, $pageSize: Int) {
    envBySlug(slug: $envSlug) {
      workflows @paginated(perPage: $pageSize, page: $page) {
        data {
          name
          id
          slug
        }
      }
    }
  }
`);

export const ScoresDashboard = ({ envSlug }: { envSlug: string }) => {
  const [selectedFns, setFns, removeFns] = useStringArraySearchParam('fns');
  const [enabledMetrics, setEnabledMetrics] =
    useStringArraySearchParam('metrics');
  const [start] = useSearchParam('start');
  const [end] = useSearchParam('end');
  const [duration] = useSearchParam('duration');
  const batchUpdate = useBatchedSearchParams();

  const parsedDuration = duration ? parseDuration(duration) : '';
  const parsedStart = toDate(start);
  const parsedEnd = toDate(end);

  const [{ data, fetching }] = useQuery({
    query: ScoresLookupDocument,
    variables: { envSlug, page: 1, pageSize: 1000 },
  });

  const functions = data?.envBySlug?.workflows.data ?? [];

  const defaultRange =
    parsedStart && parsedEnd
      ? {
          type: 'absolute' as const,
          start: parsedStart,
          end: parsedEnd,
        }
      : {
          type: 'relative' as const,
          duration: parsedDuration || DEFAULT_DURATION,
        };

  const metrics = PLACEHOLDER_METRICS;
  const enabled = enabledMetrics ?? metrics.map((m) => m.key);
  const toggleMetric = (key: string) => {
    const next = enabled.includes(key)
      ? enabled.filter((k) => k !== key)
      : [...enabled, key];
    setEnabledMetrics(next);
  };

  const visibleMetrics = metrics.filter((m) => enabled.includes(m.key));

  return (
    <div className="flex h-full w-full flex-col">
      <div className="bg-canvasBase flex flex-row items-center gap-1.5 px-3 py-[9px]">
        <TimeFilter
          // TODO: source from account log retention entitlement.
          daysAgoMax={7}
          defaultValue={defaultRange}
          onDaysChange={(range: RangeChangeProps) => {
            batchUpdate({
              duration:
                range.type === 'relative'
                  ? durationToString(range.duration)
                  : null,
              start:
                range.type === 'absolute' ? range.start.toISOString() : null,
              end: range.type === 'absolute' ? range.end.toISOString() : null,
            });
          }}
        />
        {fetching ? (
          <Skeleton className="block h-6 w-60" />
        ) : (
          <EntityFilter
            type="function"
            onFilterChange={(fns) => (fns.length ? setFns(fns) : removeFns())}
            selectedEntities={selectedFns || []}
            entities={functions}
          />
        )}
      </div>
      <div className="bg-canvasBase flex flex-row gap-4 px-4 pb-6">
        <div className="grid flex-1 grid-cols-1 gap-4 lg:grid-cols-2">
          {visibleMetrics.map((m) => (
            <ScoreCard key={m.key} metric={m} />
          ))}
        </div>
        <Legend metrics={metrics} enabled={enabled} onToggle={toggleMetric} />
      </div>
    </div>
  );
};

const ScoreCard = ({ metric }: { metric: ScoreMetric }) => {
  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col rounded-md border p-5">
      <div className="text-subtle mb-2 text-lg">{metric.label}</div>
      <div className="flex h-full items-center justify-center">
        <Skeleton className="h-full w-full" />
      </div>
    </div>
  );
};

const Legend = ({
  metrics,
  enabled,
  onToggle,
}: {
  metrics: ScoreMetric[];
  enabled: string[];
  onToggle: (key: string) => void;
}) => {
  return (
    <div className="border-subtle w-[220px] shrink-0 rounded-md border p-4">
      <div className="text-subtle mb-3 text-sm font-medium">Scores</div>
      <div className="flex flex-col gap-3">
        {metrics.map((m, i) => (
          <div key={m.key} className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span
                className="inline-block h-2.5 w-2.5 rounded-full"
                style={{
                  backgroundColor: LEGEND_COLORS[i % LEGEND_COLORS.length],
                }}
              />
              <span className="text-basis text-sm">{m.label}</span>
            </div>
            <Switch
              checked={enabled.includes(m.key)}
              onCheckedChange={() => onToggle(m.key)}
            />
          </div>
        ))}
      </div>
    </div>
  );
};
