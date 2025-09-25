import { type ReactNode } from 'react';
import { Button } from '@inngest/components/Button';
import ConfigurationBlock from '@inngest/components/FunctionConfiguration/ConfigurationBlock';
import ConfigurationCategory from '@inngest/components/FunctionConfiguration/ConfigurationCategory';
import ConfigurationSection from '@inngest/components/FunctionConfiguration/ConfigurationSection';
import ConfigurationTable, {
  type ConfigurationEntry,
} from '@inngest/components/FunctionConfiguration/ConfigurationTable';
import { PopoverContent } from '@inngest/components/FunctionConfiguration/FunctionConfigurationInfoPopovers';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill';
import { Time } from '@inngest/components/Time';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { getHumanReadableCron, useCron } from '@inngest/components/hooks/useCron';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { relativeTime } from '@inngest/components/utils/date';
import {
  RiArrowRightSLine,
  RiArrowRightUpLine,
  RiExternalLinkLine,
  RiInformationLine,
  RiTimeLine,
} from '@remixicon/react';

import type { GetFunctionQuery as DashboardGetFunctionQuery } from '../../../../apps/dashboard/src/gql/graphql';
import {
  FunctionTriggerTypes,
  type GetFunctionQuery as DevServerGetFunctionQuery,
} from '../../../../apps/dev-server-ui/src/store/generated';

type InngestFunction =
  | NonNullable<DevServerGetFunctionQuery['functionBySlug']>
  | NonNullable<DashboardGetFunctionQuery['workspace']['workflow']>;

type FunctionConfigurationProps = {
  inngestFunction: InngestFunction;
  header?: ReactNode;
  deployCreatedAt?: string | null;
  getAppLink?: () => string;
  getEventLink?: (eventName: string) => string;
  getFunctionLink?: (functionSlug: string) => string;
  getBillingUrl?: () => string;
};

export function FunctionConfiguration({
  inngestFunction,
  header,
  deployCreatedAt,
  getAppLink,
  getEventLink,
  getFunctionLink,
  getBillingUrl,
}: FunctionConfigurationProps) {
  if (!inngestFunction.configuration) {
    return null;
  }

  const configuration = inngestFunction.configuration;
  const triggers = inngestFunction.triggers;

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
              {concurrencyLimit.limit.isPlanLimit && getBillingUrl && (
                <Info
                  side="bottom"
                  align="end"
                  text={
                    <span className="whitespace-pre-line">
                      Running into limits? Easily upgrade your plan or boost concurrency on your
                      existing plan.
                    </span>
                  }
                  widthClassName="max-w-xs"
                  action={
                    <Link
                      href={getBillingUrl()}
                      target="_blank"
                      iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
                    >
                      Explore plans
                    </Link>
                  }
                  iconElement={
                    <Pill
                      className="flex items-center gap-1"
                      icon={<RiInformationLine className="h-[18px] w-[18px]" />}
                      iconSide="right"
                    >
                      <span className="whitespace-nowrap">Plan limit</span>
                    </Pill>
                  }
                />
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
    <div className="border-subtle flex h-full flex-col overflow-hidden overflow-y-auto border-l-[0.5px]">
      {header}
      <ConfigurationCategory title="Overview">
        <ConfigurationSection title="App">
          <ConfigurationBlock
            icon={<AppsIcon className="h-5 w-5" />}
            mainContent={inngestFunction.app.name}
            subContent={
              deployCreatedAt ? <Time format="relative" value={new Date(deployCreatedAt)} /> : ''
            }
            rightElement={
              getAppLink ? (
                <RiArrowRightSLine className="h-5 w-5" />
              ) : (
                <Button
                  label="Go to apps"
                  href="/apps"
                  appearance="ghost"
                  icon={<RiArrowRightUpLine />}
                  iconSide="right"
                />
              )
            }
            href={getAppLink ? getAppLink() : undefined}
          />
        </ConfigurationSection>

        <ConfigurationSection title="Triggers">
          {triggers?.map((trigger) => {
            if (trigger.type === FunctionTriggerTypes.Cron) {
              return <CronTriggerBlock key={trigger.value} schedule={trigger.value} />;
            } else if (trigger.type === FunctionTriggerTypes.Event) {
              return (
                <ConfigurationBlock
                  key={trigger.value}
                  icon={<EventsIcon className="h-5 w-5" />}
                  mainContent={trigger.value}
                  expression={trigger.condition ? `if: ${trigger.condition}` : ''}
                  rightElement={
                    getEventLink ? <RiArrowRightSLine className="h-5 w-5" /> : undefined
                  }
                  href={getEventLink ? getEventLink(trigger.value) : undefined}
                />
              );
            } else {
              // Exhaustive check - this should never be reached if all cases are handled
              const _exhaustiveCheck: never = trigger.type;
              return _exhaustiveCheck;
            }
          })}
        </ConfigurationSection>
      </ConfigurationCategory>
      <ConfigurationCategory title="Execution Configurations">
        <ConfigurationSection title="Failure Handler" infoPopoverContent={PopoverContent.failure}>
          {inngestFunction.failureHandler && (
            <ConfigurationBlock
              icon={<FunctionsIcon className="h-5 w-5" />}
              mainContent={inngestFunction.failureHandler.slug}
              rightElement={getFunctionLink ? <RiArrowRightSLine className="h-5 w-5" /> : undefined}
              href={
                getFunctionLink ? getFunctionLink(inngestFunction.failureHandler.slug) : undefined
              }
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
                rightElement={getEventLink ? <RiArrowRightSLine className="h-5 w-5" /> : undefined}
                href={getEventLink ? getEventLink(cancelOn.event) : undefined}
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

type CronTriggerBlockProps = {
  schedule: string;
};

function CronTriggerBlock({ schedule }: CronTriggerBlockProps) {
  const { nextRun } = useCron(schedule);

  return (
    <ConfigurationBlock
      icon={<RiTimeLine className="h-5 w-5" />}
      mainContent={getHumanReadableCron(schedule)}
      rightElement={<Pill className="font-mono">{schedule}</Pill>}
      subContent={
        nextRun ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="truncate">{`Next run: ${relativeTime(nextRun)}`}</span>
            </TooltipTrigger>
            <TooltipContent className="font-mono text-xs">{nextRun.toISOString()}</TooltipContent>
          </Tooltip>
        ) : (
          ''
        )
      }
    />
  );
}
