import { useMemo } from 'react';
import { ulid } from 'ulid';

import Badge from '@/components/Badge';
import SendEventButton from '@/components/Event/SendEventButton';
import MetadataGrid from '@/components/Metadata/MetadataGrid';
import { shortDate } from '@/utils/date';
import { usePrettyJson } from '../../hooks/usePrettyJson';
import { useSendEventMutation } from '../../store/devApi';
import { useGetEventQuery } from '../../store/generated';
import { selectRun } from '../../store/global';
import { useAppDispatch, useAppSelector } from '../../store/hooks';
import Button from '../Button/Button';
import CodeBlock from '../Code/CodeBlock';
import ContentCard from '../Content/ContentCard';
import FuncCard from '../Function/FuncCard';
import FuncCardFooter from '../Function/FuncCardFooter';
import { renderRunStatus } from '../Function/RunStatus';

interface EventSectionProps {
  eventId: string;
}

export const EventSection = ({ eventId }: EventSectionProps) => {
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
  const dispatch = useAppDispatch();

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

  return (
    <ContentCard
      title={event.name || 'unknown'}
      type="event"
      metadata={
        <div className="pt-8">
          <MetadataGrid
            metadataItems={[
              { label: 'Event ID', value: eventId, size: 'large' },
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
      <hr className="border-slate-800/50 mt-8" />
      <div className="px-5 py-4 gap-6 flex flex-col">
        <div className="flex items-center gap-2 pt-4">
          <h3 className="text-slate-400 text-sm">Functions</h3>
          <Badge kind="outlined">{event.functionRuns?.length.toString() || '0'}</Badge>
        </div>
        {event.functionRuns
          ?.slice()
          .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
          .map((run) => {
            const status = renderRunStatus(run);
            return (
              <FuncCard
                key={run.id}
                title={run.name || 'Unknown'}
                id={run.id}
                status={status}
                active={selectedRun === run.id}
                onClick={() => dispatch(selectRun(run.id))}
                footer={<FuncCardFooter functionRun={run} />}
              />
            );
          })}
      </div>
    </ContentCard>
  );
};
