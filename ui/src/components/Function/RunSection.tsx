import { useEffect, useMemo, useState } from "preact/hooks";
import { usePrettyJson } from "../../hooks/usePrettyJson";
import { IconFeed } from "../../icons";
import {
  EventStatus,
  FunctionEventType,
  FunctionRunStatus,
  StepEventType,
  useGetFunctionRunQuery,
} from "../../store/generated";
import Button from "../Button";
import CodeBlock from "../CodeBlock";
import ContentCard from "../Content/ContentCard";
import TimelineFuncProgress from "../Timeline/TimelineFuncProgress";
import TimelineRow from "../Timeline/TimelineRow";

interface FunctionRunSectionProps {
  runId: string | null | undefined;
}

export const FunctionRunSection = ({ runId }: FunctionRunSectionProps) => {
  const [pollingInterval, setPollingInterval] = useState(1000);
  const query = useGetFunctionRunQuery(
    { id: runId || "" },
    { pollingInterval, skip: !runId, refetchOnMountOrArgChange: true }
  );
  const run = useMemo(() => query.data?.functionRun, [query.data?.functionRun]);
  // const eventPayload = usePrettyJson(run?.event?.raw);

  useEffect(() => {
    if (typeof run?.pendingSteps !== "number") return;
    setPollingInterval(run.pendingSteps > 0 ? 1000 : 0);
  }, [run?.pendingSteps]);

  if (query.isLoading) {
    return <div>Loading...</div>;
  }

  if (!run) {
    return (
      <ContentCard title="Event not found" date={0} id="">
        Event not found
      </ContentCard>
    );
  }

  return (
    <ContentCard
      title={run.name || "Unknown"}
      date={run.startedAt}
      id={run.id}
      button={<Button label="Open Function" icon={<IconFeed />} />}
    >
      <div className="flex justify-end px-4 border-t border-slate-800/50 pt-4 mt-4">
        <Button label="Retry" />
      </div>
      <div className="pr-4 mt-4">
        {run.timeline?.map((row) => (
          <FunctionRunTimelineRow
            createdAt={row.createdAt}
            rowType={row.__typename === "FunctionEvent" ? "function" : "step"}
            eventType={
              row.__typename === "FunctionEvent"
                ? row.functionType || FunctionEventType.Completed
                : row.stepType || StepEventType.Completed
            }
            output={row.output}
          />
        ))}
      </div>
    </ContentCard>
  );
};

type FunctionRunTimelineRowProps = {
  rowType: "function" | "step";
  eventType: FunctionEventType | StepEventType;
  output: string | null | undefined;
  createdAt: string | number;
};

const FunctionRunTimelineRow = ({
  rowType,
  eventType,
  output,
  createdAt,
}: FunctionRunTimelineRowProps) => {
  const payload = usePrettyJson(output);

  const { label, status } =
    rowType === "function"
      ? functionEventTypeMap[eventType]
      : stepEventTypeMap[eventType];

  return (
    <TimelineRow status={status} iconOffset={0}>
      <TimelineFuncProgress label={label} date={createdAt} id="">
        {payload ? (
          <CodeBlock tabs={[{ label: "Payload", content: payload }]} />
        ) : null}
      </TimelineFuncProgress>
    </TimelineRow>
  );
};

const functionEventTypeMap: Record<
  FunctionEventType,
  { status: EventStatus | FunctionRunStatus; label: string }
> = {
  [FunctionEventType.Cancelled]: {
    label: "Function Cancelled",
    status: FunctionRunStatus.Cancelled,
  },
  [FunctionEventType.Completed]: {
    label: "Function Completed",
    status: FunctionRunStatus.Completed,
  },
  [FunctionEventType.Failed]: {
    label: "Function Failed",
    status: EventStatus.Failed,
  },
  [FunctionEventType.Started]: {
    label: "Function Started",
    status: EventStatus.Completed,
  },
};

const stepEventTypeMap: Record<
  StepEventType,
  { status: EventStatus | FunctionRunStatus; label: string }
> = {
  [StepEventType.Completed]: {
    label: "Step Completed",
    status: EventStatus.Completed,
  },
  [StepEventType.Failed]: { label: "Step Failed", status: EventStatus.Failed },
  [StepEventType.Started]: {
    label: "Step Started",
    status: EventStatus.Completed,
  },
  [StepEventType.Errored]: {
    label: "Step Errored",
    status: EventStatus.Failed,
  },
  [StepEventType.Scheduled]: {
    label: "Step Scheduled",
    status: EventStatus.Completed,
  },
  [StepEventType.Sleeping]: {
    label: "Step Sleeping",
    status: EventStatus.Completed,
  },
  [StepEventType.Waiting]: {
    label: "Step Waiting",
    status: EventStatus.Completed,
  },
};
