import { Header } from '@inngest/components/Header/Header';

export default function PageSkeleton({ text }: { text: string }) {
  return (
    <>
      <Header breadcrumb={[{ text }]} loading />
    </>
  );
}
