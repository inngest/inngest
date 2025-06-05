import { useMemo } from 'react';
import NextLink from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { InvokeButton } from '@inngest/components/InvokeButton';
import { MetadataGrid, type MetadataItemProps } from '@inngest/components/Metadata';
import { Pill } from '@inngest/components/Pill';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import {
  RiArrowRightSLine,
  RiArrowRightUpLine,
  RiInformationLine,
  RiTimeLine,
} from '@remixicon/react';
import { toast } from 'sonner';

import {
  FunctionTriggerTypes,
  useInvokeFunctionMutation,
  type Function,
  type FunctionConfiguration,
} from '@/store/generated';
import Block from './Block';

type FunctionConfigurationProps = {
  inngestFunction: Function;
  triggers: any;
  configuration: FunctionConfiguration;
};

export default function FunctionConfiguration({
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
    <div className="flex flex-col">
      <Header
        breadcrumb={[{ text: inngestFunction.name }]}
        action={
          <div className="flex flex-row items-center justify-end gap-2">
            <InvokeButton
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
          </div>
        }
      />
      <div className="inline-flex flex-col items-start justify-start px-4 pb-6 pt-4">
        {/* do we want 'var(--textColor-light, #9B9B9B)'*/}
        {/* letter spacing */}
        <div className="flex flex-col justify-center break-words pb-3 text-xs uppercase tracking-wide text-gray-400">
          Overview
        </div>
        <div className="flex flex-col space-y-6 self-stretch ">
          <div>
            {/* do we want font weight 450 specifically? */}
            <span className="text-sm font-medium">App</span>
            {/*border-gray-200 (#E5E7EB) would be close to #E2E2E2.*/}
            <div className="flex items-center gap-2 self-stretch rounded border border-gray-200 p-2">
              {/*bg-gray-100 (#F3F4F6) would be close to #F6F6F6.*/}
              {/*bg-[#F6F6F6]*/}
              <div className="flex h-9 w-9 items-center justify-center gap-2 rounded bg-gray-100 p-2">
                {/*width: 1.125rem → w-[1.125rem] (18px, no default utility for this size)*/}
                {/*height: 1.125rem → h-[1.125rem] (18px, no default utility for this size)*/}
                {/*flex-shrink: 0 → shrink-0*/}
                {/*aspect-ratio: 1/1 → aspect-square*/}

                {/*Note: 1.125rem = 18px, which doesn't have a default Tailwind utility (the scale goes from w-4 = 16px to w-5 = 20px), so we use the arbitrary value syntax with square brackets.*/}
                <AppsIcon className="h-5 w-5" />
              </div>
              <div className="flex grow flex-col items-start justify-center gap-1 self-stretch">
                <div>{inngestFunction.app.name}</div>
                {/*{inngestFunction.app.latestSync ? (*/}
                {/*  <div>{inngestFunction.app.latestSync}</div>*/}
                {/*) : (*/}
                {/*  <></>*/}
                {/*)}*/}
                {/*{function_.current?.deploy?.createdAt && (*/}
                {/*  <Time*/}
                {/*    className="text-subtle text-xs"*/}
                {/*    format="relative"*/}
                {/*    value={new Date(function_.current.deploy.createdAt)}*/}
                {/*  />*/}
                {/*)}*/}
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
            <span className="text-sm font-medium">Triggers</span>
            {inngestFunction.triggers.map((trigger) => (
              <div
                key={trigger.value}
                className="flex items-center gap-2 self-stretch rounded border border-gray-200 p-2"
              >
                {/*bg-gray-100 (#F3F4F6) would be close to #F6F6F6.*/}
                {/*bg-[#F6F6F6]*/}
                <div className="flex h-9 w-9 items-center justify-center gap-2 rounded bg-gray-100 p-2">
                  {/*width: 1.125rem → w-[1.125rem] (18px, no default utility for this size)*/}
                  {/*height: 1.125rem → h-[1.125rem] (18px, no default utility for this size)*/}
                  {/*flex-shrink: 0 → shrink-0*/}
                  {/*aspect-ratio: 1/1 → aspect-square*/}

                  {/*Note: 1.125rem = 18px, which doesn't have a default Tailwind utility (the scale goes from w-4 = 16px to w-5 = 20px), so we use the arbitrary value syntax with square brackets.*/}
                  {trigger.type == 'EVENT' && <EventsIcon className="h-5 w-5" />}
                  {trigger.type == 'CRON' && <RiTimeLine className="h-5 w-5" />}
                </div>
                <div className="flex grow flex-col items-start justify-center gap-1 self-stretch">
                  <div>{trigger.value}</div>
                  {trigger.type == 'EVENT' && trigger.condition && (
                    <div className="text-xs">
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
                  <div className="flex h-9 w-9 items-center justify-center gap-2 rounded bg-gray-100 p-2">
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
              <span className="text-sm font-medium">Cancellations</span>
              {inngestFunction.configuration.cancellations.map((cancelOn) => {
                // link to event in cloud
                return (
                  // className="border-subtle bg-canvasBase hover:bg-canvasMuted block rounded-md border border-gray-200 "
                  <div className="border-subtle flex items-center gap-2 self-stretch rounded border border-gray-200 p-2">
                    <div className="flex h-9 w-9 items-center justify-center gap-2 rounded bg-gray-100 p-2">
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
                        <div className="text-subtle text-xs">Timeout {cancelOn.timeout}</div>
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
                <tr className="h-8 bg-gray-100 bg-gray-50">
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
      <Block title="Configuration">
        <MetadataGrid columns={2} metadataItems={miscellaneousItems} />
        {eventBatchItems && (
          <>
            <h3 className="pb-2 pt-6 text-sm font-medium">Events Batch</h3>
            <MetadataGrid columns={2} metadataItems={eventBatchItems} />
          </>
        )}
        <h3 className="pb-2 pt-6 text-sm font-medium">Concurrency</h3>
        <div className="space-y-3">
          {configuration.concurrency.map((concurrencyItem, index) => {
            const items: MetadataItemProps[] = [
              {
                label: 'Scope',
                title: concurrencyItem.scope,
                value: (
                  <div className="lowercase first-letter:capitalize">{concurrencyItem.scope}</div>
                ),
              },
              {
                label: 'Limit',
                value: `${concurrencyItem.limit.value}`,
                ...(concurrencyItem.limit.isPlanLimit && {
                  badge: {
                    label: 'Plan Limit',
                    description:
                      'If not configured, the limit is set to the maximum value allowed within your plan.',
                  },
                }),
                tooltip: 'The maximum number of concurrently running steps.',
              },
            ];

            if (concurrencyItem.key) {
              items.push({
                label: 'Key',
                value: concurrencyItem.key,
                type: 'code',
                size: 'large',
              });
            }
            return <MetadataGrid key={index} columns={2} metadataItems={items} />;
          })}
        </div>
        {rateLimitItems && (
          <>
            <h3 className="pb-2 pt-6 text-sm font-medium">Rate Limit</h3>
            <MetadataGrid columns={2} metadataItems={rateLimitItems} />
          </>
        )}
        {debounceItems && (
          <>
            <h3 className="pb-2 pt-6 text-sm font-medium">Debounce</h3>
            <MetadataGrid columns={2} metadataItems={debounceItems} />
          </>
        )}
        {throttleItems && (
          <>
            <h3 className="pb-2 pt-6 text-sm font-medium">Throttle</h3>
            <MetadataGrid columns={2} metadataItems={throttleItems} />
          </>
        )}
      </Block>
    </div>
  );
}
