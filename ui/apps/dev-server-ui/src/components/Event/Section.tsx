import { useMemo } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Badge } from '@inngest/components/Badge';
import { Button } from '@inngest/components/Button';
import { CodeBlock } from '@inngest/components/CodeBlock';
import { ContentCard } from '@inngest/components/ContentCard';
import { FuncCard } from '@inngest/components/FuncCard';
import { FuncCardFooter } from '@inngest/components/FuncCardFooter';
import { MetadataGrid } from '@inngest/components/Metadata';
import { ulid } from 'ulid';

import SendEventButton from '@/components/Event/SendEventButton';
import { shortDate } from '@/utils/date';
import { usePrettyJson } from '../../hooks/usePrettyJson';
import { useSendEventMutation } from '../../store/devApi';
import { FunctionRunStatus, useGetEventQuery } from '../../store/generated';

interface EventSectionProps {
  eventId: string;
}

export const EventSection = ({ eventId }: EventSectionProps) => {
  const router = useRouter();
  const params = useSearchParams();
  const runID = params.get('run');
  const query = useGetEventQuery({ id: eventId }, { pollingInterval: 1500 });
  const event = useMemo(() => query.data?.event, [query.data?.event]);
  const eventPayload = usePrettyJson(event?.raw);

  const [sendEvent] = useSendEventMutation();

  if (query.isLoading) {
    return <div>Loading...</div>;
  }

  if (!event) {
    return <div>Event not found</div>;
  }

  if (!runID && event.functionRuns && event.functionRuns.length > 0) {
    const firstRunId = event.functionRuns[0]?.id;
    router.push(`/stream/trigger?event=${eventId}&run=${firstRunId}`);
  }

  return (
    <ContentCard
      title={event.name || 'unknown'}
      type="event"
      metadata={
        <div className="pt-8">
          <MetadataGrid
            metadataItems={[
              { label: 'Event ID', value: eventId, size: 'large', type: 'code' },
              { label: 'Received At', value: shortDate(new Date(event.createdAt)) },
            ]}
          />
        </div>
      }
      button={
        <div className="flex items-center gap-1">
          <Button
            label="Replay"
            btnAction={() => {
              if (!event?.raw) {
                return;
              }

              const eventId = ulid();

              sendEvent({
                ...JSON.parse(event.raw),
                id: eventId,
                ts: Date.now(),
              }).unwrap();
            }}
          />
          <SendEventButton label="Edit and Replay" appearance="outlined" data={event.raw} />
        </div>
      }
      active
    >
      {eventPayload ? (
        <div className="px-5 pt-4">
          <CodeBlock tabs={[{ label: 'Payload', content: eventPayload }]} />
        </div>
      ) : null}
      <hr className="mt-8 border-slate-800/50" />
      <div className="flex flex-col gap-6 px-5 py-4">
        <div className="flex items-center gap-2 pt-4">
          <h3 className="text-sm text-slate-400">Functions</h3>
          <Badge kind="outlined">{event.functionRuns?.length.toString() || '0'}</Badge>
        </div>
        {event.functionRuns
          ?.slice()
          .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
          .map((r) => {
            const run = {
              ...r,
              name: r.name ?? 'Unknown',
              output: r.output ?? undefined,
              status: r.status ?? FunctionRunStatus.Running,
            };

            return (
              <FuncCard
                key={run.id}
                title={run.name}
                id={run.id}
                status={run.status || undefined}
                active={runID === run.id}
                onClick={() => router.push(`/stream/trigger?event=${eventId}&run=${run.id}`)}
                footer={<FuncCardFooter functionRun={run} />}
              />
            );
          })}
      </div>
    </ContentCard>
  );
};
