import { Button } from '@inngest/components/Button';
import { CodeBlock, type CodeBlockAction } from '@inngest/components/CodeBlock';
import { ContentCard } from '@inngest/components/ContentCard';
import { FuncCard } from '@inngest/components/FuncCard';
import { MetadataGrid } from '@inngest/components/Metadata';
import { Pill } from '@inngest/components/Pill/Pill';
import { usePrettyJson } from '@inngest/components/hooks/usePrettyJson';
import type { Event } from '@inngest/components/types/event';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import { shortDate } from '@inngest/components/utils/date';

import { BatchSize } from '../BatchSize';

type EventProps = {
  batchCreatedAt?: Date | undefined;
  batchID?: string | undefined;
  events: Pick<Event, 'id' | 'name' | 'payload' | 'receivedAt'>[];
  loading?: false;
};

type LoadingEvent = {
  batchCreatedAt?: Date | undefined;
  batchID?: string | undefined;
  events?: Pick<Event, 'id' | 'name' | 'payload' | 'receivedAt'>[];
  loading: true;
};

type WithRunSelector = {
  functionRuns: Pick<FunctionRun, 'id' | 'name' | 'output' | 'status'>[];
  onFunctionRunClick: (runID: string) => void;
  onReplayEvent?: () => void;

  // TODO: Replace this with an imported component.
  SendEventButton?: React.ElementType;

  selectedRunID: string | undefined;
  codeBlockActions?: CodeBlockAction[];
};

type WithoutRunSelector = {
  functionRuns?: undefined;
  onFunctionRunClick?: undefined;
  onReplayEvent?: undefined;
  SendEventButton?: undefined;
  selectedRunID?: undefined;
  codeBlockActions?: CodeBlockAction[];
};

type Props = (EventProps | LoadingEvent) & (WithoutRunSelector | WithRunSelector);

export function EventDetails({
  batchCreatedAt,
  batchID,
  events,
  functionRuns,
  onFunctionRunClick,
  onReplayEvent,
  selectedRunID,
  SendEventButton,
  codeBlockActions = [],
  loading = false,
}: Props) {
  let singleEvent: NonNullable<typeof events>[number] | undefined = undefined;
  if (!batchID && events?.length === 1) {
    singleEvent = events[0];
  }

  let batch = undefined;
  if (batchID) {
    batch = events;
  }

  const isInternalEvent = Boolean(singleEvent?.name?.startsWith('inngest/'));
  let prettyPayload = undefined;
  if (singleEvent && singleEvent.payload) {
    prettyPayload = usePrettyJson(singleEvent.payload);
  } else if (batch) {
    prettyPayload = usePrettyJson(
      JSON.stringify(
        batch.map((e) => {
          return JSON.parse(e.payload);
        })
      )
    );
  }

  if (batch && functionRuns) {
    // For batches, we only want to show the selected run. This is because each
    // run gets its own batch -- batches aren't shared between runs
    functionRuns = functionRuns.filter((run) => run.id === selectedRunID);
  }

  return (
    <ContentCard
      title={events?.[0]?.name || 'unknown'}
      type="event"
      metadata={
        <>
          {singleEvent && (
            <div className="pt-8">
              <MetadataGrid
                metadataItems={[
                  { label: 'Event ID', value: singleEvent.id, size: 'large', type: 'code' },
                  {
                    label: 'Received At',
                    value: shortDate(singleEvent.receivedAt),
                  },
                ]}
                loading={loading}
              />
            </div>
          )}
          {batch && (
            <div className="pt-8">
              <MetadataGrid
                metadataItems={[
                  { label: 'Batch ID', value: batchID ?? '-', size: 'large', type: 'code' },
                  {
                    label: 'Created At',
                    value: batchCreatedAt ? shortDate(batchCreatedAt) : '-',
                  },
                ]}
                loading={loading}
              />
            </div>
          )}
        </>
      }
      button={
        <>
          {!isInternalEvent && onReplayEvent && SendEventButton && (
            <>
              <div className="flex items-center gap-1">
                <Button
                  label="Replay"
                  kind="secondary"
                  appearance="outlined"
                  onClick={onReplayEvent}
                />
                <SendEventButton />
              </div>
            </>
          )}
          {batch && events && <BatchSize eventCount={events.length} />}
        </>
      }
      active
    >
      {!loading && (
        <div className="px-5 pt-4">
          <CodeBlock.Wrapper>
            <CodeBlock
              header={{ title: batch ? 'Batch' : 'Payload' }}
              tab={{ content: prettyPayload ?? 'Unknown' }}
              actions={codeBlockActions}
            />
          </CodeBlock.Wrapper>
        </div>
      )}

      {functionRuns && onFunctionRunClick && (
        <>
          <hr className="border-subtle mt-8" />
          <div className="flex flex-col gap-6 px-5 py-4">
            <div className="flex items-center gap-2 pt-4">
              <h3 className="text-subtle text-sm">Functions</h3>
              <Pill appearance="outlined">{functionRuns.length.toString() || '0'}</Pill>
            </div>
            {functionRuns
              .slice()
              .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
              .map((run) => {
                return (
                  <FuncCard
                    key={run.id}
                    title={run.name}
                    id={run.id}
                    status={run.status}
                    active={selectedRunID === run.id}
                    onClick={() => onFunctionRunClick(run.id)}
                  />
                );
              })}
          </div>
        </>
      )}
    </ContentCard>
  );
}
