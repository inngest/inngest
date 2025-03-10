import type React from 'react';
import { AccordionList, AccordionPrimitive } from '@inngest/components/AccordionCard/AccordionList';
import DescriptionListItem from '@inngest/components/Apps/DescriptionListItem';
import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton';
import { Time } from '@inngest/components/Time';
import { RiArrowDownSLine, RiArrowLeftRightLine, RiInfinityLine } from '@remixicon/react';

import { Card } from '../Card';
import { methodTypes, type App } from '../types/app';
import { cn } from '../utils/classNames';

type CardKind = 'default' | 'warning' | 'primary' | 'error' | 'info';

const kindStyles = {
  primary: {
    background: 'bg-primary-moderate',
    text: 'text-primary-intense',
  },
  error: { background: 'bg-tertiary-moderate', text: 'text-tertiary-moderate' },
  warning: { background: 'bg-accent-moderate', text: 'text-accent-moderate' },
  default: { background: 'bg-surfaceSubtle', text: 'text-surfaceSubtle' },
  info: { background: 'bg-secondary-moderate', text: 'text-secondary-intense' },
};

export function SkeletonCard() {
  return (
    <Card>
      <div className="text-basis mb-px ml-1 flex flex-col justify-between p-6">
        <div className="mb-6">
          <div className="pb-3">
            <Skeleton className="mb-2 block h-8 w-96" />
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-x-12 gap-y-6 pt-1.5">
          <DescriptionListItem
            term="Last synced at"
            detail={<Skeleton className="block h-5 w-36" />}
          />
          <DescriptionListItem term="Method" detail={<Skeleton className="block h-5 w-20" />} />
          <DescriptionListItem
            term="SDK version"
            detail={<Skeleton className="block h-5 w-12" />}
          />
          <DescriptionListItem term="Language" detail={<Skeleton className="block h-5 w-20" />} />
          <DescriptionListItem term="Framework" detail={<Skeleton className="block h-5 w-20" />} />
        </div>
      </div>
      <div className="border-muted border-t px-6 py-3">
        <Skeleton className="block h-6 w-28" />
      </div>
    </Card>
  );
}

// Cards used in the Apps main list
export function AppCard({ kind, children }: React.PropsWithChildren<{ kind: CardKind }>) {
  return (
    <Card accentColor={kindStyles[kind].background} accentPosition="left">
      {children}
    </Card>
  );
}

type CardContentProps = {
  app: App;
  pill: React.ReactNode;
  actions: React.ReactNode;
  url?: string;
  workerCounter?: React.ReactNode;
};

export function AppCardContent({ url, app, pill, actions, workerCounter }: CardContentProps) {
  const Wrapper = url ? 'a' : 'div';

  return (
    <div className="text-basis flex flex-col justify-between p-6">
      <div className="mb-6">
        <div className="items-top flex justify-between">
          <Wrapper className="inline text-xl" href={url}>
            {app.name}
            {pill ? <span className="ml-2 inline-flex align-middle">{pill}</span> : null}
          </Wrapper>
          {actions}
        </div>
        <p className="text-muted mt-0.5 truncate">
          {app.url ||
            (app.appVersion && (
              <>
                Current app version: <span className="text-basis">{app.appVersion}</span>
              </>
            ))}
        </p>
      </div>

      <dl className="flex flex-wrap items-center gap-x-12 gap-y-6">
        {app.lastSyncedAt && (
          <DescriptionListItem term="Last synced at" detail={<Time value={app.lastSyncedAt} />} />
        )}
        <DescriptionListItem
          term="Method"
          detail={
            <div className="flex items-center gap-1">
              {app.method === methodTypes.Connect ? (
                <RiInfinityLine className="h-4 w-4" />
              ) : (
                <RiArrowLeftRightLine className="h-4 w-4" />
              )}
              <div className="lowercase first-letter:capitalize">{app.method}</div>
            </div>
          }
        />
        <DescriptionListItem
          term="SDK version"
          detail={app.sdkVersion ? <Pill>{app.sdkVersion}</Pill> : '-'}
        />
        <DescriptionListItem term="Language" detail={app.sdkLanguage ?? '-'} />
        {app.method === methodTypes.Connect ? (
          <>{workerCounter}</>
        ) : (
          <DescriptionListItem term="Framework" detail={app.framework ?? '-'} />
        )}
      </dl>
    </div>
  );
}

type CardFooterProps = {
  kind: CardKind;
  headerTitle: React.ReactNode;
  headerSecondaryCTA: React.ReactNode;
  content: React.ReactNode;
};

export function AppCardFooter({ kind, headerTitle, headerSecondaryCTA, content }: CardFooterProps) {
  return (
    <AccordionList
      type="multiple"
      defaultValue={[]}
      className="border-subtle rounded-none border-0 border-t"
    >
      <AccordionList.Item value="description" className="px-4 py-3">
        <AccordionPrimitive.Header
          className={cn('flex items-center gap-1 text-sm', kindStyles[kind].text)}
        >
          <div className="flex w-full items-center justify-between">
            <AccordionPrimitive.Trigger asChild>
              <span className="group flex items-center gap-1 text-sm font-medium">
                <Button
                  appearance="ghost"
                  kind="secondary"
                  size="small"
                  icon={
                    <RiArrowDownSLine
                      className={cn(
                        'transform-90 transition-transform duration-500 group-data-[state=open]:-rotate-180',
                        kindStyles[kind].text
                      )}
                    />
                  }
                />
                {headerTitle}
              </span>
            </AccordionPrimitive.Trigger>
            {headerSecondaryCTA}
          </div>
        </AccordionPrimitive.Header>
        <AccordionList.Content className="px-7">{content}</AccordionList.Content>
      </AccordionList.Item>
    </AccordionList>
  );
}

AppCard.Content = AppCardContent;
AppCard.Footer = AppCardFooter;
