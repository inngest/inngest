import { useMemo } from 'react';
import NextLink from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
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
  type FunctionConfiguration,
} from '../../../../apps/dev-server-ui/src/store/generated';

type FunctionConfigurationProps = {
  onClose: () => void;
  inngestFunction: Function;
  triggers: any;
  configuration: FunctionConfiguration;
};

export function FunctionConfiguration({
  onClose,
  inngestFunction,
  triggers,
  configuration,
}: FunctionConfigurationProps) {
  const router = useRouter();
  const doesFunctionAcceptPayload = useMemo(() => {
    return Boolean(triggers?.some((trigger) => trigger.type === FunctionTriggerTypes.Event));
  }, [triggers]);

  const [invokeFunction] = useInvokeFunctionMutation();

  const miscellaneousItems: MetadataItemProps[] = [
    {
      size: 'large',
      label: 'Retries',
      value: configuration.retries.value.toString(),
      ...(configuration.retries.isDefault && {
        badge: {
          label: 'Default',
        },
      }),
      tooltip: 'The number of times the function will be retried when it errors.',
    },
  ];

  if (configuration.priority) {
    miscellaneousItems.push({
      label: 'Priority',
      value: configuration.priority,
      size: 'large',
      type: 'code',
      tooltip:
        'When the function is triggered multiple times simultaneously, the priority determines the order in which they are run.' +
        '\n\n' +
        'The priority value is determined by evaluating the configured expression. The higher the value, the higher the priority.',
    });
  }

  let rateLimitItems: MetadataItemProps[] | undefined;
  if (configuration.rateLimit) {
    rateLimitItems = [
      {
        label: 'Period',
        value: configuration.rateLimit.period,
      },
      { label: 'Limit', value: configuration.rateLimit.limit.toString() },
    ];

    if (configuration.rateLimit.key) {
      rateLimitItems.push({
        label: 'Key',
        value: configuration.rateLimit.key,
        type: 'code',
        size: 'large',
      });
    }
  }

  let debounceItems: MetadataItemProps[] | undefined;
  if (configuration.debounce) {
    debounceItems = [
      {
        label: 'Period',
        value: configuration.debounce.period,
      },
    ];

    if (configuration.debounce.key) {
      debounceItems.push({
        label: 'Key',
        value: configuration.debounce.key,
        type: 'code',
      });
    }
  }

  let eventBatchItems: MetadataItemProps[] | undefined;
  if (configuration.eventsBatch) {
    eventBatchItems = [
      {
        label: 'Max Size',
        value: configuration.eventsBatch.maxSize.toString(),
      },
      {
        label: 'Timeout',
        value: configuration.eventsBatch.timeout,
      },
    ];

    if (configuration.eventsBatch.key) {
      eventBatchItems.push({
        label: 'Key',
        value: configuration.eventsBatch.key,
        type: 'code',
      });
    }
  }

  let throttleItems: MetadataItemProps[] | undefined;
  if (configuration.throttle) {
    throttleItems = [
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
      throttleItems.push({
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
      <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
        {/* letter spacing */}
        <h2 className="text-light pb-3 text-xs font-medium uppercase leading-4 tracking-wider">
          Overview
        </h2>
        <div className="flex flex-col space-y-6 self-stretch ">
          <div>
            {/* do we want font weight 450 specifically? */}
            <h3 className="text-basis mb-1 text-sm font-medium">App</h3>
            {/*border-gray-200 (#E5E7EB) would be close to #E2E2E2.*/}
            <div className="border-subtle flex items-center gap-2 self-stretch rounded border p-2">
              <div className="bg-canvasSubtle text-light flex h-9 w-9 items-center justify-center gap-2 rounded p-2">
                <AppsIcon className="h-5 w-5" />
              </div>
              <div className="text-basis flex grow flex-col items-start justify-center gap-1 self-stretch text-sm font-medium">
                <div>{inngestFunction.app.name}</div>
              </div>
              <div className="self-end">
                <Button
                  label="Go to apps"
                  href="/apps"
                  appearance="ghost"
                  icon={<RiArrowRightUpLine />}
                  iconSide="right"
                />
              </div>
            </div>
          </div>

          <div>
            <h3 className="text-basis mb-1 text-sm font-medium">Triggers</h3>
            <div>
              {inngestFunction.triggers.map((trigger) => (
                <div
                  key={trigger.value}
                  className="border-subtle flex items-center gap-2 self-stretch border border-b-0 p-2 first:rounded-t last:rounded-b last:border-b"
                >
                  <div className="bg-canvasSubtle text-light flex h-9 w-9 items-center justify-center gap-2 rounded p-2">
                    {trigger.type == 'EVENT' && <EventsIcon className="h-5 w-5" />}
                    {trigger.type == 'CRON' && <RiTimeLine className="h-5 w-5" />}
                  </div>
                  <div className="text-basis flex grow flex-col items-start justify-center gap-1 self-stretch text-sm font-medium">
                    <div>{trigger.value}</div>
                    {trigger.type == 'EVENT' && trigger.condition && (
                      <div className="text-muted text-sm">
                        <code>if: {trigger.condition}</code>
                        {/*handle overflow and pop up*/}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
      <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
        <div className="flex flex-col justify-center break-words pb-3 text-xs uppercase tracking-wide text-gray-400">
          Execution Configurations
        </div>
        <div className="flex flex-col space-y-6 self-stretch ">
          {/* should this one be even if it's not set, educational? */}
          {inngestFunction.failureHandler && (
            <div>
              <span className="text-sm font-medium">Failure Handler</span>
              <NextLink
                // link does not work yet, return to once we have finalized link structure
                href={`/functions/config?slug=${inngestFunction.failureHandler.slug}`}
                className="border-subtle bg-canvasBase hover:bg-canvasMuted block rounded-md border border-gray-200 "
              >
                <div className="flex items-center gap-2 self-stretch rounded p-2">
                  <div className="flex h-9 w-9 items-center justify-center gap-2 rounded bg-gray-100 p-2 dark:bg-transparent">
                    <FunctionsIcon className="h-5 w-5" />
                  </div>
                  <div className="flex grow flex-col items-start justify-center gap-1 self-stretch">
                    <div>{inngestFunction.failureHandler.slug}</div>
                  </div>
                  <RiArrowRightSLine className="h-5" />
                </div>
              </NextLink>
            </div>
          )}
          {inngestFunction.configuration.cancellations && (
            <div>
              <span className="text-sm font-medium">Cancel On</span>
              {inngestFunction.configuration.cancellations.map((cancelOn) => {
                // link to event in cloud
                return (
                  // className="border-subtle bg-canvasBase hover:bg-canvasMuted block rounded-md border border-gray-200 "
                  <div className="border-subtle flex items-center gap-2 self-stretch rounded border border-gray-200 p-2">
                    <div className="flex h-9 w-9 items-center justify-center gap-2 rounded bg-gray-100 p-2 dark:bg-transparent">
                      <EventsIcon className="h-5 w-5" />
                    </div>
                    <div className="flex grow flex-col items-start justify-center gap-1 self-stretch">
                      <div>{cancelOn.event}</div>
                      {cancelOn.condition && (
                        <div className="text-xs">
                          <code>if: {cancelOn.condition}</code>
                          {/*handle overflow and pop up*/}
                        </div>
                      )}
                      {cancelOn.timeout && (
                        <div className="text-subtle text-xs">Timeout: {cancelOn.timeout}</div>
                      )}
                    </div>
                    {/*<RiArrowRightSLine className="h-5" />*/}
                  </div>
                );
              })}
            </div>
          )}

          <div className="overflow-hidden rounded border border-gray-300 ">
            <table className="w-full border-collapse">
              <thead>
                <tr className="h-8 border-b bg-gray-100 dark:bg-transparent">
                  <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
                    <div className="flex items-center gap-2">
                      Retries
                      <RiInformationLine className="h-5 w-5" />
                    </div>
                  </td>
                </tr>
              </thead>
              <tbody>
                {/*can't apply px-2 to tr*/}
                <tr className="h-8 border-b border-gray-200">
                  <td className="text-muted px-2 text-sm">Value</td>
                  <td className="text-basis px-2 text-right text-sm">
                    {inngestFunction.configuration.retries.value} retries
                    {/*fix pluralization*/}
                    {inngestFunction.configuration.retries.isDefault && (
                      <Pill className="ml-2">Default</Pill>
                    )}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
        <div className="flex flex-col justify-center break-words pb-3 text-xs uppercase tracking-wide text-gray-400">
          Scheduling Configurations
        </div>
        <div className="flex flex-col space-y-6 self-stretch ">
          {inngestFunction.configuration.rateLimit && (
            <div className="overflow-hidden rounded border border-gray-300 ">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="h-8 border-b bg-gray-100 dark:bg-transparent">
                    <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
                      <div className="flex items-center gap-2">
                        Rate limit
                        <RiInformationLine className="h-5 w-5" />
                      </div>
                    </td>
                  </tr>
                </thead>
                <tbody>
                  {/*can't apply px-2 to tr*/}
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Limit</td>
                    <td className="text-basis px-2 text-right text-sm">
                      {inngestFunction.configuration.rateLimit.limit.toString()}
                    </td>
                  </tr>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Period</td>
                    <td className="text-basis px-2 text-right text-sm">
                      {inngestFunction.configuration.rateLimit.period}
                    </td>
                  </tr>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Key</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.rateLimit.key}</code>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          )}
          {inngestFunction.configuration.debounce && (
            <div className="overflow-hidden rounded border border-gray-300 ">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="h-8 border-b bg-gray-100 dark:bg-transparent">
                    <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
                      <div className="flex items-center gap-2">
                        Debounce
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
                      {inngestFunction.configuration.debounce.period}
                    </td>
                  </tr>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Key</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.debounce.key}</code>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          )}
          {inngestFunction.configuration.priority && (
            <div className="overflow-hidden rounded border border-gray-300 ">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="h-8 border-b bg-gray-100 dark:bg-transparent">
                    <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
                      <div className="flex items-center gap-2">
                        Priority
                        <RiInformationLine className="h-5 w-5" />
                      </div>
                    </td>
                  </tr>
                </thead>
                <tbody>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Run</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.priority}</code>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          )}
          {inngestFunction.configuration.eventsBatch && (
            <div className="overflow-hidden rounded border border-gray-300 ">
              <table className="w-full border-collapse">
                <thead>
                  <tr className="h-8 border-b bg-gray-100 dark:bg-transparent">
                    <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
                      <div className="flex items-center gap-2">
                        Batching
                        <RiInformationLine className="h-5 w-5" />
                      </div>
                    </td>
                  </tr>
                </thead>
                <tbody>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Max size</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.eventsBatch.maxSize.toString()}</code>
                    </td>
                  </tr>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Timeout</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.eventsBatch.timeout}</code>
                    </td>
                  </tr>
                  <tr className="h-8 border-b border-gray-200">
                    <td className="text-muted px-2 text-sm">Key</td>
                    <td className="text-basis px-2 text-right text-sm">
                      <code>{inngestFunction.configuration.eventsBatch.key}</code>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
      <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
        <div className="flex flex-col justify-center break-words pb-3 text-xs uppercase tracking-wide text-gray-400">
          Queue Configurations
        </div>
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
      </div>
    </div>
  );
}
