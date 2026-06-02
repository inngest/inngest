import { RiBarChartBoxLine } from '@remixicon/react';

type EmptyStateReason = 'not-configured' | 'no-data' | 'no-plottable-data';

const messages: Record<
  EmptyStateReason,
  { header: string; subheader: string }
> = {
  'not-configured': {
    header: 'No charts configured',
    subheader: 'Please choose from the available options to create a chart.',
  },
  'no-data': {
    header: 'No data to chart',
    subheader: 'Run a query first to get data for charting.',
  },
  'no-plottable-data': {
    header: 'Cannot plot this data',
    subheader: 'The selected columns cannot be converted to plottable values.',
  },
};

export function InsightsChartEmptyState({
  reason,
}: {
  reason: EmptyStateReason;
}) {
  const { header, subheader } = messages[reason];

  return (
    <div className="flex h-full flex-col items-center justify-center gap-4">
      <div className="flex max-w-[410px] flex-col items-center gap-4">
        <div className="bg-canvasSubtle flex h-[56px] w-[56px] items-center justify-center rounded-lg p-3">
          <RiBarChartBoxLine className="text-light h-6 w-6" />
        </div>
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-basis text-xl font-medium">{header}</h3>
          <p className="text-muted text-sm">{subheader}</p>
        </div>
      </div>
    </div>
  );
}
