import { Button } from '@inngest/components/Button';
import { Skeleton } from '@inngest/components/Skeleton';
import * as Collapsible from '@radix-ui/react-collapsible';
import { RiContractRightFill, RiExpandLeftFill } from '@remixicon/react';
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

  return (
    <Collapsible.Root
      className={cn(showEventPanel && 'w-2/6', 'flex flex-col gap-5', className)}
      open={showEventPanel}
      onOpenChange={setShowEventPanel}
    >
      {!showEventPanel && (
        <Collapsible.Trigger asChild>
          <button>
            <RiExpandLeftFill className="text-slate-500" />
          </button>
        </Collapsible.Trigger>
      )}
      <Collapsible.Content>
        {/* <div className='data-[state=open]:animate-slideRight data-[state=closed]:animate-slideLeft overflow-hidden'> */}
        <Card className="rounded-r-none">
          <Card.Header className="h-11 flex-row items-center gap-2">
            <div className="flex grow items-center gap-2">Trigger details</div>
            <Collapsible.Trigger asChild>
              <Button size="large" appearance="text" icon={<RiContractRightFill />} />
            </Collapsible.Trigger>
          </Card.Header>

          <Card.Content>
            <div>
              <dl className="flex flex-wrap gap-4">
                <Labeled label="Trigger Name">
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
        {/* </div> */}
      </Collapsible.Content>
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
