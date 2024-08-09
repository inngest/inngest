import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import PageSkeleton from '@/components/Skeleton/PageSkeleton';

export default async function Loading() {
  const newIANav = await getBooleanFlag('new-ia-nav');
  return newIANav ? <PageSkeleton text="Event Search" /> : '';
}
