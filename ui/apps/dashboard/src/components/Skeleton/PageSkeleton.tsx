import { Header } from '@inngest/components/Header/Header';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';

export default function PageSkeleton({ text }: { text: string }) {
  return (
    <>
      <Header breadcrumb={[{ text }]} />
      <Skeleton className="mx-4 mt-6 h-32 w-full" />
    </>
  );
}
