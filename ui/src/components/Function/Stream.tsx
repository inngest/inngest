import {
  FunctionRunStatus,
  useGetFunctionsStreamQuery,
} from "../../store/generated";
import { selectEvent, selectRun } from "../../store/global";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import TimelineFeedContent from "../Timeline/TimelineFeedContent";
import TimelineRow from "../Timeline/TimelineRow";

export const FuncStream = () => {
  const functions = useGetFunctionsStreamQuery(
    {},
    { pollingInterval: 1000, refetchOnMountOrArgChange: true }
  );
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
  const dispatch = useAppDispatch();

  return (
    <>
      {functions.data?.functionRuns?.map((run, i, list) => (
        <TimelineRow
          key={run.id}
          status={run.status || FunctionRunStatus.Completed}
          iconOffset={30}
          topLine={i !== 0}
          bottomLine={i < list.length - 1}
        >
          <TimelineFeedContent
            date={run.startedAt}
            active={selectedRun === run.id}
            status={run.status || FunctionRunStatus.Completed}
            badge={run.pendingSteps || 0}
            name={run.name || "Unknown"}
            onClick={() => {
              dispatch(selectRun(run.id));
              dispatch(selectEvent(run.event?.id || null));
            }}
          />
        </TimelineRow>
      ))}
    </>
  );
};
