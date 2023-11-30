import { Badge } from '@inngest/components/Badge/Badge';
import { Button } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock';
import { ContentCard } from '@inngest/components/ContentCard';
import { FuncCard } from '@inngest/components/FuncCard';
import { FuncCardFooter } from '@inngest/components/FuncCardFooter';
import { MetadataGrid } from '@inngest/components/Metadata';
import { usePrettyJson } from '@inngest/components/hooks/usePrettyJson';
import type { Event } from '@inngest/components/types/event';
import type { FunctionRun } from '@inngest/components/types/functionRun';
import { shortDate } from '@inngest/components/utils/date';

type EventProps = {
  event: Pick<Event, 'id' | 'name' | 'payload' | 'receivedAt'>;
  loading?: false;
};

type LoadingEvent = {
  event: Partial<Event>;
  loading: true;
};

type WithRunSelector = {
  functionRuns: Pick<FunctionRun, 'id' | 'name' | 'output' | 'status'>[];
  onFunctionRunClick: (runID: string) => void;
  onReplayEvent?: () => void;

  // TODO: Replace this with an imported component.
  SendEventButton?: React.ElementType;

  selectedRunID: string | undefined;
};

type WithoutRunSelector = {
  functionRuns?: undefined;
  onFunctionRunClick?: undefined;
  onReplayEvent?: undefined;
  SendEventButton?: undefined;
  selectedRunID?: undefined;
};

type Props = (EventProps | LoadingEvent) & (WithoutRunSelector | WithRunSelector);

export function EventDetails({
  event,
  functionRuns,
  onFunctionRunClick,
  onReplayEvent,
  selectedRunID,
  SendEventButton,
  loading = false,
}: Props) {
  const prettyPayload = usePrettyJson(event.payload);

  return (
    <ContentCard
      title={event.name || 'unknown'}
      type="event"
      metadata={
        <div className="pt-8">
          <MetadataGrid
            metadataItems={[
              { label: 'Event ID', value: event.id || '', size: 'large', type: 'code' },
              {
                label: 'Received At',
                value: (event.receivedAt && shortDate(event.receivedAt)) || '',
              },
            ]}
            loading={loading}
          />
        </div>
      }
      button={
        onReplayEvent &&
        SendEventButton && (
          <>
            <div className="flex items-center gap-1">
              <Button label="Replay" btnAction={onReplayEvent} />
              <SendEventButton />
            </div>
          </>
        )
      }
      active
    >
      {!loading && (
        <div className="px-5 pt-4">
          <CodeBlock tabs={[{ label: 'Payload', content: prettyPayload ?? 'Unknown' }]} />
        </div>
      )}

      {functionRuns && onFunctionRunClick && (
        <>
          <hr className="mt-8 border-slate-800/50" />
          <div className="flex flex-col gap-6 px-5 py-4">
            <div className="flex items-center gap-2 pt-4">
              <h3 className="text-sm text-slate-400">Functions</h3>
              <Badge kind="outlined">{functionRuns.length.toString() || '0'}</Badge>
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
                    footer={<FuncCardFooter functionRun={run} />}
                  />
                );
              })}
          </div>
        </>
      )}
    </ContentCard>
  );
}
