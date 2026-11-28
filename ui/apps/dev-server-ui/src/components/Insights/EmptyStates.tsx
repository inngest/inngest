import { RiTableView } from '@remixicon/react';

// Mirrors the dashboard Insights feature's InsightsDataTable/states/IconLayoutWrapper.tsx,
// EmptyState.tsx, and ResultsState/NoResults.tsx (minus the examples-link action, which is
// gated behind a Cloud-only temp flag).
function IconLayoutWrapper({
  header,
  subheader,
}: {
  header: string;
  subheader: string;
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-4">
      <div className="flex max-w-[410px] flex-col items-center gap-4">
        <div className="bg-canvasSubtle flex h-[56px] w-[56px] items-center justify-center rounded-lg p-3">
          <RiTableView className="text-light h-6 w-6" />
        </div>
        <div className="flex flex-col gap-2 text-center">
          <h3 className="text-basis text-xl font-medium">{header}</h3>
          <p className="text-muted text-sm">{subheader}</p>
        </div>
      </div>
    </div>
  );
}

export function QueryEmptyState() {
  return (
    <IconLayoutWrapper
      header="Your query results will appear here"
      subheader="Run a query to analyze your data and the results will be displayed here."
    />
  );
}

export function NoResultsState() {
  return (
    <IconLayoutWrapper
      header="No results found"
      subheader="We couldn't find any results matching your search. Try adjusting your query."
    />
  );
}
