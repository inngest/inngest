import type React from 'react';
import { Time } from '@inngest/components/Time';

import { type App } from '../types/app';
import { cn } from '../utils/classNames';

type CardKind = 'default' | 'warning' | 'primary' | 'error' | 'info';

const kindStyles = {
  primary: 'bg-primary-moderate',
  error: 'bg-tertiary-moderate',
  warning: 'bg-accent-moderate',
  default: 'bg-surfaceSubtle',
  info: 'bg-secondary-moderate',
};

export function AppCard({ kind, children }: React.PropsWithChildren<{ kind: CardKind }>) {
  return (
    <div className="border-subtle bg-canvasBase relative max-w-3xl rounded border">
      <div
        className={cn('absolute bottom-0 left-0 top-0 w-1 rounded-l-[0.2rem]', kindStyles[kind])}
      />
      {children}
    </div>
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
        <Description term="Sync method" detail={app.syncMethod} />
        <Description term="SDK version" detail={app.sdkVersion} />
        <Description term="Language" detail={app.language} />
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
      <dd className="text-subtle text-sm">{detail ?? ''}</dd>
    </div>
  );
}
