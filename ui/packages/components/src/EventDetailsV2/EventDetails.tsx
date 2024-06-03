import { Button } from '@inngest/components/Button';
import { Skeleton } from '@inngest/components/Skeleton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import * as Collapsible from '@radix-ui/react-collapsible';
import { RiContractRightFill, RiExpandLeftFill } from '@remixicon/react';
import { AnimatePresence, motion } from 'framer-motion';
import { useLocalStorage } from 'react-use';

import { Card } from '../Card';
import { CodeBlock } from '../CodeBlock';
import { Time } from '../Time';
import { cn } from '../utils/classNames';

type Props = {
  isLoading: boolean;
  className?: string;
  trigger: {
    output?: string;
    name: string;
    receivedAt: string;
    id: string;
  };
};

export function EventDetails({ isLoading, className, trigger }: Props) {
  const [showEventPanel, setShowEventPanel] = useLocalStorage('showEventPanel', true);
  /* TODO: Exit animation before unmounting */
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
                <div className="flex grow items-center gap-2">Event details</div>
                <Collapsible.Trigger asChild>
                  <Button size="large" appearance="text" icon={<RiContractRightFill />} />
                </Collapsible.Trigger>
              </Card.Header>

              <Card.Content>
                <div>
                  <dl className="flex flex-wrap gap-4">
                    <Labeled label="Event Name">
                      {isLoading ? <Skeleton className="h-5 w-full" /> : trigger.name}
                    </Labeled>
                    <Labeled label="Event ID">
                      <span className="font-mono">
                        {isLoading ? <Skeleton className="h-5 w-full" /> : trigger.id}
                      </span>
                    </Labeled>

                    <Labeled label="Received at">
                      {isLoading ? (
                        <Skeleton className="h-5 w-full" />
                      ) : (
                        <Time value={new Date(trigger.receivedAt)} />
                      )}
                    </Labeled>
                  </dl>
                </div>
              </Card.Content>
            </Card>

            {trigger.output && (
              <CodeBlock
                tabs={[
                  {
                    label: 'Payload',
                    content: trigger.output,
                  },
                ]}
              />
            )}
          </motion.div>
        </Collapsible.Content>
      </AnimatePresence>
    </Collapsible.Root>
  );
}

function Labeled({ label, children }: React.PropsWithChildren<{ label: string }>) {
  return (
    <div className="w-64 text-sm">
      <dt className="pb-2 text-slate-500">{label}</dt>
      <dd className="truncate">{children}</dd>
    </div>
  );
}
