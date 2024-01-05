'use client';

import { Squares2X2Icon } from '@heroicons/react/20/solid';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import Header, { type HeaderLink } from '@/components/Header/Header';
import { useBooleanSearchParam } from '@/utils/useSearchParam';
import { Apps } from './Apps';

export default function Page() {
  const env = useEnvironment();

  const [isArchived] = useBooleanSearchParam('archived');

  const navLinks: HeaderLink[] = [
    {
      active: isArchived !== true,
      href: `/env/${env.slug}/apps`,
      text: 'Active',
    },
    {
      active: isArchived,
      href: `/env/${env.slug}/apps?archived=true`,
      text: 'Archived',
    },
  ];

  return (
    <>
      <Header
        icon={<Squares2X2Icon className="h-5 w-5 text-white" />}
        links={navLinks}
        title="Apps"
      />
      <div className="h-full overflow-y-auto bg-slate-100">
        <Apps isArchived={isArchived} />
      </div>
    </>
  );
}
