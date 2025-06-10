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
import { type MetadataItemProps } from '@inngest/components/Metadata';
import { Pill } from '@inngest/components/Pill';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import {
  RiArrowRightSLine,
  RiArrowRightUpLine,
  RiCloseLine,
  RiInformationLine,
  RiTimeLine,
} from '@remixicon/react';
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

  let throttleEntries: MetadataItemProps[] | undefined;
  if (configuration.throttle) {
    throttleEntries = [
      {
        label: 'Period',
        value: configuration.throttle.period,
      },
      {
        label: 'Limit',
        value: configuration.throttle.limit.toString(),
      },
      {
        label: 'Burst',
        value: configuration.throttle.burst.toString(),
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
        <div className="flex flex-col space-y-6 self-stretch ">
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
                    <div className="text-muted text-sm">
                      <code>if: {trigger.condition}</code>
                      {/*handle overflow and pop up*/}
                    </div>
                  ) : (
                    <></>
                  )
                }
              />
            ))}
          </ConfigurationSection>
        </div>
      </ConfigurationCategory>
      <ConfigurationCategory title="Execution Configurations">
        <div className="flex flex-col space-y-6 self-stretch ">
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
                    <>
                      {cancelOn.condition && (
                        <div className="text-xs">
                          <code>if: {cancelOn.condition}</code>
                          {/*handle overflow and pop up*/}
                        </div>
                      )}
                      {cancelOn.timeout && (
                        <div className="text-subtle text-xs">Timeout: {cancelOn.timeout}</div>
                      )}
                    </>
                  }
                />
              );
            })}
          </ConfigurationSection>

          <ConfigurationTable header="Retries" entries={retryEntries} />
        </div>
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
        <div className="flex flex-col space-y-6 self-stretch ">
          {inngestFunction.configuration.concurrency &&
            inngestFunction.configuration.concurrency.map((concurrencyConfig, index) => (
              <div className="overflow-hidden rounded border border-gray-300 ">
                <table className="w-full border-collapse">
                  <thead>
                    <tr className="h-8 border-b bg-gray-100 dark:bg-transparent">
                      <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
                        <div className="flex items-center gap-2">
                          Concurrency ({index + 1})
                          <RiInformationLine className="h-5 w-5" />
                        </div>
                      </td>
                    </tr>
                  </thead>
                  <tbody>
                    {/*can't apply px-2 to tr*/}
                    <tr className="h-8 border-b border-gray-200">
                      <td className="text-muted px-2 text-sm">Scope</td>
                      <td className="text-basis px-2 text-right text-sm">
                        {concurrencyConfig.scope}
                      </td>
                    </tr>
                    <tr className="h-8 border-b border-gray-200">
                      <td className="text-muted px-2 text-sm">Limit</td>
                      <td className="text-basis px-2 text-right text-sm">
                        {concurrencyConfig.limit.value}
                        {concurrencyConfig.limit.isPlanLimit && (
                          <Pill className="ml-2">Plan limit</Pill>
                        )}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            ))}
          {inngestFunction.configuration.throttle && (
            <div className="overflow-hidden rounded border border-gray-300 ">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="h-8 border-b bg-gray-100 dark:bg-transparent">
                    <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
                      <div className="flex items-center gap-2">
                        Throttle
                        <RiInformationLine className="h-5 w-5" />
                      </div>
                    </td>
                  </tr>
                </thead>
                <tbody>
                  {/*can't apply px-2 to tr*/}
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Period</td>
                    <td className="text-basis px-2 text-right text-sm">
                      {inngestFunction.configuration.throttle.period}
                    </td>
                  </tr>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Limit</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.throttle.limit}</code>
                    </td>
                  </tr>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Burst</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.throttle.burst}</code>
                    </td>
                  </tr>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Key</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.throttle.key}</code>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          )}
        </div>
      </ConfigurationCategory>
    </div>
  );
}
