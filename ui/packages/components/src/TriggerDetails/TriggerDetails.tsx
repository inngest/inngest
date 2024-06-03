import { Button } from '@inngest/components/Button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { usePrettyJson } from '@inngest/components/hooks/usePrettyJson';
import * as Collapsible from '@radix-ui/react-collapsible';
import { RiContractRightFill, RiExpandLeftFill } from '@remixicon/react';
import { AnimatePresence, motion } from 'framer-motion';
import { useLocalStorage } from 'react-use';

import { Card } from '../Card';
import { CodeBlock } from '../CodeBlock';
import {
  ElementWrapper,
  IDElement,
  SkeletonElement,
  TextElement,
  TimeElement,
} from '../DetailsCard/Element';
import { cn } from '../utils/classNames';

type Props = {
  isLoading?: false;
  className?: string;
  trigger: {
    payloads: string[];
    timestamp: string;
    eventName: string | null;
    IDs: string[];
    batchID: string | null;
    isBatch: boolean;
    cron: string | null;
  };
};

type LoadingProps = {
  isLoading: true;
  trigger?: undefined;
  className?: string;
};

export function TriggerDetails({ isLoading, className, trigger }: Props | LoadingProps) {
  const [showEventPanel, setShowEventPanel] = useLocalStorage('showEventPanel', true);
  /* TODO: Exit animation before unmounting */

  let type = 'EVENT';
  if (trigger?.isBatch) {
    type = 'BATCH';
  } else if (trigger?.cron) {
    type = 'CRON';
  }

  let prettyPayload = undefined;
  if (trigger?.payloads) {
    if (trigger.payloads.length === 1 && trigger.payloads[0]) {
      prettyPayload = usePrettyJson(trigger.payloads[0]);
    } else {
      prettyPayload = usePrettyJson(
        JSON.stringify(
          trigger.payloads.map((e) => {
            return JSON.parse(e);
          })
        )
      );
    }
  }

  return (
    <Collapsible.Root
      className={cn(showEventPanel && 'w-2/5', 'flex flex-col gap-5', className)}
      open={showEventPanel}
      onOpenChange={setShowEventPanel}
    >
      <AnimatePresence>
        {!showEventPanel && (
          <Collapsible.Trigger asChild>
            <button className="flex h-8 w-8 items-center justify-center rounded-full border border-slate-400">
              <Tooltip>
                <TooltipTrigger>
                  <RiExpandLeftFill className="text-slate-400" />
                </TooltipTrigger>
                <TooltipContent>Show event details</TooltipContent>
              </Tooltip>
            </button>
          </Collapsible.Trigger>
        )}
        <Collapsible.Content>
          <motion.div
            className=""
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{
              duration: 0.5,
              type: 'tween',
            }}
          >
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
                    {type === 'CRON' && (
                      <>
                        <ElementWrapper label="Cron expression">
                          {isLoading ? <SkeletonElement /> : <IDElement>{trigger?.cron}</IDElement>}
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
                            <TextElement>{trigger.eventName}</TextElement>
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
              </Card.Content>
            </Card>

            {trigger?.payloads && type !== 'CRON' && (
              <div className="mt-4">
                <CodeBlock
                  tabs={[
                    {
                      label: trigger.isBatch ? 'Batch' : 'Event Payload',
                      content: prettyPayload ?? 'Unknown',
                    },
                  ]}
                />
              </div>
            )}
          </motion.div>
        </Collapsible.Content>
      </AnimatePresence>
    </Collapsible.Root>
  );
}
