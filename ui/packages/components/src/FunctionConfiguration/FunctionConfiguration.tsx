import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import ConfigurationBlock from '@inngest/components/FunctionConfiguration/ConfigurationBlock';
import ConfigurationCategory from '@inngest/components/FunctionConfiguration/ConfigurationCategory';
import ConfigurationSection from '@inngest/components/FunctionConfiguration/ConfigurationSection';
import ConfigurationTable, {
  type ConfigurationEntry,
} from '@inngest/components/FunctionConfiguration/ConfigurationTable';
import { PopoverContent } from '@inngest/components/FunctionConfiguration/FunctionConfigurationInfoPopovers';
import { Header } from '@inngest/components/Header/Header';
import { InvokeButton } from '@inngest/components/InvokeButton';
import { Pill } from '@inngest/components/Pill';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { RiArrowRightSLine, RiArrowRightUpLine, RiCloseLine, RiTimeLine } from '@remixicon/react';
import { toast } from 'sonner';

import {
  FunctionTriggerTypes,
  useInvokeFunctionMutation,
  type GetFunctionQuery,
} from '../../../../apps/dev-server-ui/src/store/generated';

type FunctionConfigurationProps = {
  onClose: () => void;
  inngestFunction: NonNullable<GetFunctionQuery['functionBySlug']>;
};

export function FunctionConfiguration({ onClose, inngestFunction }: FunctionConfigurationProps) {
  const configuration = inngestFunction.configuration;
  const triggers = inngestFunction.triggers;

  const router = useRouter();
  const doesFunctionAcceptPayload = useMemo(() => {
    return Boolean(triggers?.some((trigger) => trigger.type === FunctionTriggerTypes.Event));
  }, [triggers]);

  const [invokeFunction] = useInvokeFunctionMutation();

  const retryEntries: ConfigurationEntry[] = [
    {
      label: 'Value',
      value: (
        <>
          {inngestFunction.configuration.retries.value}{' '}
          {inngestFunction.configuration.retries.value == 1 ? 'retry' : 'retries'}
          {inngestFunction.configuration.retries.isDefault && <Pill className="ml-2">Default</Pill>}
        </>
      ),
    },
  ];

  let rateLimitEntries: ConfigurationEntry[] = [];
  if (configuration.rateLimit) {
    rateLimitEntries = [
      { label: 'Limit', value: configuration.rateLimit.limit },
      {
        label: 'Period',
        value: configuration.rateLimit.period,
      },
    ];

    if (configuration.rateLimit.key) {
      rateLimitEntries.push({
        label: 'Key',
        value: configuration.rateLimit.key,
        type: 'code',
      });
    }
  }

  let debounceEntries: ConfigurationEntry[] = [];
  if (configuration.debounce) {
    debounceEntries = [
      {
        label: 'Period',
        value: configuration.debounce.period,
      },
    ];

    if (configuration.debounce.key) {
      debounceEntries.push({
        label: 'Key',
        value: configuration.debounce.key,
        type: 'code',
      });
    }
  }

  const priorityEntries: ConfigurationEntry[] = [
    {
      label: 'Run',
      value: configuration.priority,
      type: 'code',
    },
  ];

  let eventBatchEntries: ConfigurationEntry[] = [];
  if (configuration.eventsBatch) {
    eventBatchEntries = [
      {
        label: 'Max size',
        value: configuration.eventsBatch.maxSize,
      },
      {
        label: 'Timeout',
        value: configuration.eventsBatch.timeout,
      },
    ];

    if (configuration.eventsBatch.key) {
      eventBatchEntries.push({
        label: 'Key',
        value: configuration.eventsBatch.key,
        type: 'code',
      });
    }
  }

  let singletonEntries: ConfigurationEntry[] = [];
  if (configuration.singleton) {
    singletonEntries = [
      {
        label: 'Mode',
        value: configuration.singleton.mode,
      },
    ];

    if (configuration.singleton.key) {
      singletonEntries.push({
        label: 'Key',
        value: configuration.singleton.key,
        type: 'code',
      });
    }
  }

  let throttleEntries: ConfigurationEntry[] = [];
  if (configuration.throttle) {
    throttleEntries = [
      {
        label: 'Period',
        value: configuration.throttle.period,
      },
      {
        label: 'Limit',
        value: configuration.throttle.limit,
      },
      {
        label: 'Burst',
        value: configuration.throttle.burst,
      },
    ];

    if (configuration.throttle.key) {
      throttleEntries.push({
        label: 'Key',
        value: configuration.throttle.key,
        type: 'code',
      });
    }
  }

  // We show a separate table per concurrency configuration and only want to show the count if there is more than one
  const concurrencyLimitCount = configuration.concurrency.length;
  const concurrencyLimits: ConfigurationEntry[][] = configuration.concurrency.map(
    (concurrencyLimit) => {
      let concurrencyEntries: ConfigurationEntry[] = [
        {
          label: 'Scope',
          value: concurrencyLimit.scope,
        },
        {
          label: 'Limit',
          value: (
            <>
              {concurrencyLimit.limit.value >= 1 && concurrencyLimit.limit.value}
              {concurrencyLimit.limit.isPlanLimit && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span>
                      <Pill className="ml-2">Plan limit</Pill>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    If not configured, the limit is set to the maximum value allowed within your
                    plan.
                  </TooltipContent>
                </Tooltip>
              )}
            </>
          ),
        },
      ];
      if (concurrencyLimit.key) {
        concurrencyEntries.push({
          label: 'Key',
          value: concurrencyLimit.key,
          type: 'code',
        });
      }
      return concurrencyEntries;
    }
  );

  return (
    <div className="border-subtle flex flex-col overflow-hidden overflow-y-auto border-l-[0.5px]">
      <Header
        breadcrumb={[{ text: inngestFunction.name }]}
        action={
          <div className="flex flex-row items-center justify-end gap-2">
            <InvokeButton
              kind="primary"
              appearance="solid"
              disabled={false}
              doesFunctionAcceptPayload={doesFunctionAcceptPayload}
              btnAction={async ({ data, user }) => {
                await invokeFunction({
                  data,
                  functionSlug: inngestFunction.slug,
                  user,
                });
                toast.success('Function invoked');
                router.push('/runs');
              }}
            />
            <Button
              icon={<RiCloseLine className="text-muted h-5 w-5" />}
              kind="secondary"
              appearance="ghost"
              size="small"
              onClick={() => onClose()}
            />
          </div>
        }
      />
      <ConfigurationCategory title="Overview">
        <ConfigurationSection title="App">
          <ConfigurationBlock
            icon={<AppsIcon className="h-5 w-5" />}
            mainContent={inngestFunction.app.name}
            rightElement={
              <Button
                label="Go to apps"
                href="/apps"
                appearance="ghost"
                icon={<RiArrowRightUpLine />}
                iconSide="right"
              />
            }
          />
        </ConfigurationSection>

        <ConfigurationSection title="Triggers">
          {triggers?.map((trigger) => (
            <ConfigurationBlock
              key={trigger.value}
              icon={
                trigger.type == FunctionTriggerTypes.Event ? (
                  <EventsIcon className="h-5 w-5" />
                ) : (
                  <RiTimeLine className="h-5 w-5" />
                )
              }
              mainContent={trigger.value}
              expression={
                trigger.type == FunctionTriggerTypes.Event && trigger.condition
                  ? `if: ${trigger.condition}`
                  : ''
              }
            />
          ))}
        </ConfigurationSection>
      </ConfigurationCategory>
      <ConfigurationCategory title="Execution Configurations">
        <ConfigurationSection title="Failure Handler" infoPopoverContent={PopoverContent.failure}>
          {inngestFunction.failureHandler && (
            <ConfigurationBlock
              icon={<FunctionsIcon className="h-5 w-5" />}
              mainContent={inngestFunction.failureHandler.slug}
              rightElement={<RiArrowRightSLine className="h-5 w-5" />}
              href={`/functions/config?slug=${inngestFunction.failureHandler.slug}`}
            />
          )}
        </ConfigurationSection>

        <ConfigurationSection title="Cancel On" infoPopoverContent={PopoverContent.cancelOn}>
          {inngestFunction.configuration.cancellations.map((cancelOn) => {
            return (
              <ConfigurationBlock
                key={cancelOn.event}
                icon={<EventsIcon className="h-5 w-5" />}
                mainContent={cancelOn.event}
                subContent={cancelOn.timeout ? `Timeout: ${cancelOn.timeout}` : ''}
                expression={cancelOn.condition ? `if: ${cancelOn.condition}` : ''}
              />
            );
          })}
        </ConfigurationSection>

        <ConfigurationTable
          header="Retries"
          entries={retryEntries}
          infoPopoverContent={PopoverContent.retries}
        />
      </ConfigurationCategory>
      <ConfigurationCategory title="Scheduling Configurations">
        {inngestFunction.configuration.rateLimit && (
          <ConfigurationTable
            header="Rate limit"
            entries={rateLimitEntries}
            infoPopoverContent={PopoverContent.rateLimit}
          />
        )}
        {inngestFunction.configuration.debounce && (
          <ConfigurationTable
            header="Debounce"
            entries={debounceEntries}
            infoPopoverContent={PopoverContent.debounce}
          />
        )}
        {inngestFunction.configuration.priority && (
          <ConfigurationTable
            header="Priority"
            entries={priorityEntries}
            infoPopoverContent={PopoverContent.priority}
          />
        )}
        {inngestFunction.configuration.eventsBatch && (
          <ConfigurationTable
            header="Batching"
            entries={eventBatchEntries}
            infoPopoverContent={PopoverContent.batching}
          />
        )}
        {inngestFunction.configuration.singleton && (
          <ConfigurationTable
            header="Singleton"
            entries={singletonEntries}
            infoPopoverContent={PopoverContent.singleton}
          />
        )}
      </ConfigurationCategory>
      <ConfigurationCategory title="Queue Configurations">
        {inngestFunction.configuration.concurrency &&
          concurrencyLimits.map((concurrencyLimit, index) => {
            const header = concurrencyLimitCount > 1 ? `Concurrency (${index + 1})` : 'Concurrency';
            return (
              <ConfigurationTable
                key={index}
                header={header}
                entries={concurrencyLimit}
                infoPopoverContent={PopoverContent.concurrency}
              />
            );
          })}
        {inngestFunction.configuration.throttle && (
          <ConfigurationTable
            header="Throttle"
            entries={throttleEntries}
            infoPopoverContent={PopoverContent.throttle}
          />
        )}
      </ConfigurationCategory>
    </div>
  );
}
