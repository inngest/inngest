import { useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import ConfigurationBlock from '@inngest/components/FunctionConfiguration/ConfigurationBlock';
import ConfigurationCategory from '@inngest/components/FunctionConfiguration/ConfigurationCategory';
import ConfigurationSection from '@inngest/components/FunctionConfiguration/ConfigurationSection';
import ConfigurationTable, {
  type ConfigurationEntry,
} from '@inngest/components/FunctionConfiguration/ConfigurationTable';
import { Header } from '@inngest/components/Header/Header';
import { InvokeButton } from '@inngest/components/InvokeButton';
import { Pill } from '@inngest/components/Pill';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { RiArrowRightSLine, RiArrowRightUpLine, RiCloseLine, RiTimeLine } from '@remixicon/react';
import { toast } from 'sonner';

import {
  FunctionTriggerTypes,
  useInvokeFunctionMutation,
  type Function,
} from '../../../../apps/dev-server-ui/src/store/generated';

type FunctionConfigurationProps = {
  onClose: () => void;
  inngestFunction: Function;
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
      // fix pluralization
      label: 'Value',
      value: (
        <>
          {inngestFunction.configuration.retries.value} retries
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
        value: <code>{configuration.rateLimit.key}</code>,
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
        value: <code>{configuration.debounce.key}</code>,
      });
    }
  }

  const priorityEntries: ConfigurationEntry[] = [
    {
      label: 'Run',
      value: <code>{configuration.priority}</code>,
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
        value: <code>{configuration.eventsBatch.key}</code>,
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
        value: <code>{configuration.singleton.key}</code>,
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
        value: <code>{configuration.throttle.key}</code>,
      });
    }
  }

  // We show a separate table per concurrency configuration and only want to show the count if there is more than one
  const concurrencyLimitCount = configuration.concurrency.length;
  const concurrencyLimits: ConfigurationEntry[][] = configuration.concurrency.map(
    (concurrencyLimit) => {
      let concurrencyEntries = [
        {
          label: 'Scope',
          value: concurrencyLimit.scope,
        },
        {
          label: 'Limit',
          value: (
            <>
              {concurrencyLimit.limit.value >= 1 && concurrencyLimit.limit.value}
              {concurrencyLimit.limit.isPlanLimit && <Pill className="ml-2">Plan limit</Pill>}
            </>
          ),
        },
      ];
      if (concurrencyLimit.key) {
        concurrencyEntries.push({
          label: 'Key',
          value: <code>{concurrencyLimit.key}</code>,
        });
      }
      return concurrencyEntries;
    }
  );

  return (
    <div className="flex flex-col overflow-hidden overflow-y-auto">
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
            mainText={inngestFunction.app.name}
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
                trigger.type == 'EVENT' ? (
                  <EventsIcon className="h-5 w-5" />
                ) : (
                  <RiTimeLine className="h-5 w-5" />
                )
              }
              mainText={trigger.value}
              subText={
                trigger.type == 'EVENT' && trigger.condition ? (
                  <div>
                    <code className="font-mono">if: {trigger.condition}</code>
                  </div>
                ) : (
                  <></>
                )
              }
            />
          ))}
        </ConfigurationSection>
      </ConfigurationCategory>
      <ConfigurationCategory title="Execution Configurations">
        <ConfigurationSection title="Failure Handler">
          {inngestFunction.failureHandler && (
            <ConfigurationBlock
              icon={<FunctionsIcon className="h-5 w-5" />}
              mainText={inngestFunction.failureHandler.slug}
              rightElement={<RiArrowRightSLine className="h-5 w-5" />}
              href={`/functions/config?slug=${inngestFunction.failureHandler.slug}`}
            />
          )}
        </ConfigurationSection>

        <ConfigurationSection title="Cancel On">
          {inngestFunction.configuration.cancellations.map((cancelOn) => {
            return (
              <ConfigurationBlock
                key={cancelOn.event}
                icon={<EventsIcon className="h-5 w-5" />}
                mainText={cancelOn.event}
                subText={
                  <div>
                    {cancelOn.condition && (
                      <div className="text-muted text-xs">
                        <code className="font-mono">if: {cancelOn.condition}</code>
                      </div>
                    )}
                    {cancelOn.timeout && (
                      <div className="text-muted text-xs">Timeout: {cancelOn.timeout}</div>
                    )}
                  </div>
                }
              />
            );
          })}
        </ConfigurationSection>

        <ConfigurationTable header="Retries" entries={retryEntries} />
      </ConfigurationCategory>
      <ConfigurationCategory title="Scheduling Configurations">
        {inngestFunction.configuration.rateLimit && (
          <ConfigurationTable header="Rate limit" entries={rateLimitEntries} />
        )}
        {inngestFunction.configuration.debounce && (
          <ConfigurationTable header="Debounce" entries={debounceEntries} />
        )}
        {inngestFunction.configuration.priority && (
          <ConfigurationTable header="Priority" entries={priorityEntries} />
        )}
        {inngestFunction.configuration.eventsBatch && (
          <ConfigurationTable header="Batching" entries={eventBatchEntries} />
        )}
        {inngestFunction.configuration.singleton && (
          <ConfigurationTable header="Singleton" entries={singletonEntries} />
        )}
      </ConfigurationCategory>
      <ConfigurationCategory title="Queue Configurations">
        {inngestFunction.configuration.concurrency &&
          concurrencyLimits.map((concurrencyLimit, index) => {
            const header = concurrencyLimitCount > 1 ? `Concurrency (${index + 1})` : 'Concurrency';
            return <ConfigurationTable header={header} entries={concurrencyLimit} />;
          })}
        {inngestFunction.configuration.throttle && (
          <ConfigurationTable header="Throttle" entries={throttleEntries} />
        )}
      </ConfigurationCategory>
    </div>
  );
}
