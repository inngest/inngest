import type React from 'react';
import { AccordionList, AccordionPrimitive } from '@inngest/components/AccordionCard/AccordionList';
import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton';
import { Time } from '@inngest/components/Time';
import { RiArrowDownSLine, RiArrowLeftRightLine, RiInfinityLine } from '@remixicon/react';

import { Card } from '../Card';
import { connectionTypes, type App } from '../types/app';
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
      <div className="text-basis mb-px ml-1 p-6">
        <div className="mb-6">
          <div className="pb-3">
            <Skeleton className="mb-2 block h-8 w-96" />
          </div>
        </div>

        <div className="grid grid-cols-5 gap-5 pt-1.5">
          <Description term="Last synced at" detail={<Skeleton className="block h-5 w-36" />} />
          <Description term="Sync method" detail={<Skeleton className="block h-5 w-28" />} />
          <Description term="SDK version" detail={<Skeleton className="block h-5 w-14" />} />
          <Description term="Language" detail={<Skeleton className="block h-5 w-28" />} />
          <Description term="Framework" detail={<Skeleton className="block h-5 w-28" />} />
        </div>
      </div>
      <div className="border-muted border-t px-6 py-3">
        <Skeleton className="block h-6 w-28" />
      </div>
    </Card>
  );
}

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
};

export function AppCardContent({ app, pill, actions }: CardContentProps) {
  return (
    <div className="text-basis p-6">
      <div className="mb-6">
        <div className="items-top flex justify-between">
          <div className="flex items-center gap-2 text-xl">
            {app.name}
            {pill}
          </div>
          {actions}
        </div>
        <p className="text-muted mt-0.5">{app.url}</p>
      </div>

      <div className="grid grid-cols-5 gap-4">
        {app.lastSyncedAt && (
          <Description term="Last synced at" detail={<Time value={app.lastSyncedAt} />} />
        )}
        <Description
          term="Sync method"
          detail={
            <div className="flex items-center gap-1">
              {app.connectionType === connectionTypes.Connect ? (
                <RiInfinityLine className="h-4 w-4" />
              ) : (
                <RiArrowLeftRightLine className="h-4 w-4" />
              )}
              <div className="lowercase first-letter:capitalize">{app.connectionType}</div>
            </div>
          }
        />
        <Description
          term="SDK version"
          detail={app.sdkVersion?.trim() ? <Pill>{app.sdkVersion}</Pill> : '-'}
        />
        <Description term="Language" detail={app.sdkLanguage?.trim() ? app.sdkLanguage : '-'} />
        {/* TODO: Add Connected Workers counter */}
        {app.connectionType === connectionTypes.Connect ? null : (
          <Description term="Framework" detail={app.framework ?? '-'} />
        )}
      </div>
    </div>
  );
}

function Description({
  className,
  detail,
  term,
}: {
  className?: string;
  detail: React.ReactNode;
  term: string;
}) {
  return (
    <div className={className}>
      <dt className="text-light pb-1 text-sm">{term}</dt>
      <dd className="text-subtle truncate text-sm">{detail}</dd>
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
      className="border-muted rounded-none border-0 border-t"
    >
      <AccordionList.Item value="description" className="px-4 py-3">
        <AccordionPrimitive.Header
          className={cn('flex items-center gap-1 text-sm', kindStyles[kind].text)}
        >
          <div className="flex w-full items-center justify-between">
            <AccordionPrimitive.Trigger asChild>
              <span className="group flex items-center gap-1 text-sm font-medium">
                <Button
                  className="h-6 p-1"
                  appearance="ghost"
                  kind="secondary"
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
