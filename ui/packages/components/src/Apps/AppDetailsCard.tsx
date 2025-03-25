'use client';

import DescriptionListItem from '@inngest/components/Apps/DescriptionListItem';
import { cn } from '@inngest/components/utils/classNames';

type Props = {
  title: string;
  className?: string;
};

// Cards used in each app's details page and on sync pages
export default function AppDetailsCard({
  title,
  className,
  children,
}: React.PropsWithChildren<Props>) {
  return (
    <>
      <div className={cn('border-subtle bg-canvasSubtle rounded-md border', className)}>
        <h2 className="text-muted border-subtle border-b px-6 py-3 text-sm">{title}</h2>

        <dl className="bg-canvasBase flex flex-col gap-4 rounded-b-md p-6 md:grid md:grid-cols-4">
          {children}
        </dl>
      </div>
    </>
  );
}

AppDetailsCard.Item = DescriptionListItem;
