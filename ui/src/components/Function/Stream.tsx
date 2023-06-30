import { FunctionRunStatus, useGetFunctionsStreamQuery } from '../../store/generated';
import TimelineFeedContent from '../Timeline/TimelineFeedContent';
import TimelineRow from '../Timeline/TimelineRow';

export const FuncStream = () => {
  const functions = useGetFunctionsStreamQuery(undefined, { pollingInterval: 1500 });

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
            status={run.status || FunctionRunStatus.Completed}
            badge={run.pendingSteps || 0}
            name={run.name || 'Unknown'}
            active={false}
            href=""
          />
        </TimelineRow>
      ))}
    </>
  );
};
