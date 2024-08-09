import { Skeleton } from '@inngest/components/Skeleton/Skeleton';

import { Header } from '../Header/Header';

export default function PageSkeleton({ text }: { text: string }) {
  return (
    <>
      <Header breadcrumb={[{ text }]} />
      <Skeleton className="mx-4 mt-6 h-32 w-full" />
    </>
  );
}
