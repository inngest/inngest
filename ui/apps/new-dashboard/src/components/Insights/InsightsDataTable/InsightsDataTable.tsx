import { useInsightsStateMachineContext } from "../InsightsStateMachineContext/InsightsStateMachineContext";
import { EmptyState } from "./states/EmptyState";
import { ErrorState } from "./states/ErrorState";
import { LoadingState } from "./states/LoadingState";
import { ResultsState } from "./states/ResultsState/ResultsState";

export function InsightsDataTable() {
  const { status } = useInsightsStateMachineContext();

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      {(() => {
        switch (status) {
          case "error":
            return <ErrorState />;
          case "initial":
            return <EmptyState />;
          case "loading":
            return <LoadingState />;
          case "success": {
            return <ResultsState />;
          }
        }
      })()}
    </div>
  );
}
