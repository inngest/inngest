import { useEffect, useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import * as Collapsible from '@radix-ui/react-collapsible';
import { RiContractRightFill, RiExpandLeftFill } from '@remixicon/react';
import { useLocalStorage } from 'react-use';

import { Card } from '../Card';
import { CodeBlock } from '../CodeBlock';
import {
  CodeElement,
  ElementWrapper,
  IDElement,
  SkeletonElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { cn } from '../utils/classNames';

type Props = {
  className?: string;
  getTrigger: () => Promise<Trigger>;
};

export type Trigger = {
  payloads: string[];
  timestamp: string;
  eventName: string | null;
  IDs: string[];
  batchID: string | null;
  isBatch: boolean;
  cron: string | null;
};

export function TriggerDetails({ className, getTrigger }: Props) {
  const [showEventPanel, setShowEventPanel] = useLocalStorage('showEventPanel', true);
  const [trigger, setTrigger] = useState<Trigger>();
  const isLoading = !trigger;

  useEffect(() => {
    getTrigger().then((data) => {
      setTrigger(data);
    });
  }, [getTrigger]);

  const prettyPayload = useMemo(() => {
    if (!trigger?.payloads) return null;
    let payload = 'Unknown';
    if (trigger.payloads.length === 1 && trigger.payloads[0]) {
      payload = trigger.payloads[0];
    } else if (trigger.payloads.length > 1) {
      payload = JSON.stringify(
        trigger.payloads.map((e) => {
          return JSON.parse(e);
        })
      );
    }
    try {
      const data = JSON.parse(payload);
      if (data === null) {
        throw new Error();
      }

      return JSON.stringify(data, null, 2);
    } catch (e) {
      console.warn('Unable to parse content as JSON: ', payload);
      return '';
    }
  }, [trigger?.payloads]);

  let type = 'EVENT';
  if (trigger?.isBatch) {
    type = 'BATCH';
  } else if (trigger?.cron) {
    type = 'CRON';
  }

  return (
    <Collapsible.Root
      className={cn(showEventPanel && 'w-2/5', 'flex flex-col gap-5', className)}
      open={showEventPanel}
      onOpenChange={setShowEventPanel}
    >
      {!showEventPanel && (
        <Collapsible.Trigger asChild>
          <button className="flex h-8 w-8 items-center justify-center rounded-full border border-slate-400">
            <Tooltip>
              <TooltipTrigger>
                <RiExpandLeftFill className="text-slate-400" />
              </TooltipTrigger>
              <TooltipContent>Show trigger details</TooltipContent>
            </Tooltip>
          </button>
        </Collapsible.Trigger>
      )}
      <Collapsible.Content>
        {showEventPanel && (
          <>
            <Card>
              <Card.Header className="h-11 flex-row items-center gap-2">
                <div className="flex grow items-center gap-2">Trigger details</div>
                <Collapsible.Trigger asChild>
                  <Button size="large" appearance="text" icon={<RiContractRightFill />} />
                </Collapsible.Trigger>
              </Card.Header>

              <Card.Content>
                <div>
                  <dl className="flex flex-wrap gap-4">
                    {type === 'EVENT' && (
                      <>
                        <ElementWrapper label="Event name">
                          {isLoading ? (
                            <SkeletonElement />
                          ) : (
                            <TextElement>{trigger.eventName}</TextElement>
                          )}
                        </ElementWrapper>
                        <ElementWrapper label="Event ID">
                          {isLoading ? (
                            <SkeletonElement />
                          ) : (
                            <IDElement>{trigger.IDs[0]}</IDElement>
                          )}
                        </ElementWrapper>
                        <ElementWrapper label="Received at">
                          {isLoading ? (
                            <SkeletonElement />
                          ) : (
                            <TimeElement date={new Date(trigger.timestamp)} />
                          )}
                        </ElementWrapper>
                      </>
                    )}
                    {type === 'CRON' && trigger?.cron && (
                      <>
                        <ElementWrapper label="Cron expression">
                          {isLoading ? <SkeletonElement /> : <CodeElement value={trigger.cron} />}
                        </ElementWrapper>
                        <ElementWrapper label="Cron ID">
                          {isLoading ? (
                            <SkeletonElement />
                          ) : (
                            <IDElement>{trigger.IDs[0]}</IDElement>
                          )}
                        </ElementWrapper>
                        <ElementWrapper label="Triggered at">
                          {isLoading ? (
                            <SkeletonElement />
                          ) : (
                            <TimeElement date={new Date(trigger.timestamp)} />
                          )}
                        </ElementWrapper>
                      </>
                    )}
                    {type === 'BATCH' && (
                      <>
                        <ElementWrapper label="Event name">
                          {isLoading ? (
                            <SkeletonElement />
                          ) : (
                            <TextElement>{trigger.eventName ?? '-'}</TextElement>
                          )}
                        </ElementWrapper>
                        <ElementWrapper label="Batch ID">
                          {isLoading ? (
                            <SkeletonElement />
                          ) : (
                            <IDElement>{trigger.batchID}</IDElement>
                          )}
                        </ElementWrapper>
                        <ElementWrapper label="Received at">
                          {isLoading ? (
                            <SkeletonElement />
                          ) : (
                            <TimeElement date={new Date(trigger.timestamp)} />
                          )}
                        </ElementWrapper>
                      </>
                    )}
                  </dl>
                </div>

                {trigger?.payloads && type !== 'CRON' && (
                  <div className="mt-4">
                    <CodeBlock
                      tabs={[
                        {
                          label: trigger.isBatch ? 'Batch' : 'Event payload',
                          content: prettyPayload ?? 'Unknown',
                        },
                      ]}
                    />
                  </div>
                )}
              </Card.Content>
            </Card>
          </>
        )}
      </Collapsible.Content>
    </Collapsible.Root>
  );
}
