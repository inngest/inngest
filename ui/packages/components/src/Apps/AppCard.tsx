import type React from 'react';
import { AccordionList, AccordionPrimitive } from '@inngest/components/AccordionCard/AccordionList';
import { Button } from '@inngest/components/Button';
import { Time } from '@inngest/components/Time';
import { RiArrowDownSLine } from '@remixicon/react';

import { Card } from '../Card';
import { type App } from '../types/app';
import { cn } from '../utils/classNames';

type CardKind = 'default' | 'warning' | 'primary' | 'error' | 'info';

const kindStyles = {
  primary: {
    background: 'bg-primary-moderate',
    text: 'text-primary-moderate',
  },
  error: { background: 'bg-tertiary-moderate', text: 'text-tertiary-moderate' },
  warning: { background: 'bg-accent-moderate', text: 'text-accent-moderate' },
  default: { background: 'bg-surfaceSubtle', text: 'text-surfaceSubtle' },
  info: { background: 'bg-secondary-moderate', text: 'text-secondary-moderate' },
};

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
        <p className="text-subtle mt-0.5">{app.url}</p>
      </div>

      <div className="flex justify-between">
        {app.lastSyncedAt && (
          <Description term="Last synced at" detail={<Time value={app.lastSyncedAt} />} />
        )}
        <Description
          term="Sync method"
          detail={<div className="lowercase first-letter:capitalize">{app.connectionType}</div>}
        />
        <Description term="SDK version" detail={app.sdkVersion?.trim() ? app.sdkVersion : '-'} />
        <Description term="Language" detail={app.sdkLanguage?.trim() ? app.sdkLanguage : '-'} />
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
      <dd className="text-subtle text-sm">{detail}</dd>
    </div>
  );
}

type CardFooterProps = {
  kind: CardKind;
  header: React.ReactNode;
  content: React.ReactNode;
};

export function AppCardFooter({ kind, header, content }: CardFooterProps) {
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
          <AccordionPrimitive.Trigger asChild>
            <Button
              className="group h-6 p-1"
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
          </AccordionPrimitive.Trigger>
          {header}
        </AccordionPrimitive.Header>
        <AccordionList.Content className="px-7">{content}</AccordionList.Content>
      </AccordionList.Item>
    </AccordionList>
  );
}

AppCard.Content = AppCardContent;
AppCard.Footer = AppCardFooter;
