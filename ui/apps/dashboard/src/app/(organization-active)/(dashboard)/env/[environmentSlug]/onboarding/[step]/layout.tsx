'use client';

import dynamic from 'next/dynamic';
import { useRouter } from 'next/navigation';

import PageHeader from '@/components/Onboarding/PageHeader';
import { isValidStep } from '@/components/Onboarding/types';
import { pathCreator } from '@/utils/urls';

// Disable SSR in Menu, to prevent hydration errors. It requires windows info
const Menu = dynamic(() => import('@/components/Onboarding/Menu'), {
  ssr: false,
});

export default function Layout({
  children,
  params: { environmentSlug: envSlug, step },
}: React.PropsWithChildren<{ params: { environmentSlug: string; step: string } }>) {
  const router = useRouter();
  if (!isValidStep(step)) {
    router.push(pathCreator.onboarding());
    return;
  }
  return (
    <div className="text-basis my-12 grid grid-cols-3">
      <main className="col-span-2 mx-20">
        <PageHeader stepName={step} />
        {children}
      </main>
      <Menu envSlug={envSlug} stepName={step} />
    </div>
  );
}
