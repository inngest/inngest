'use client';

import { useBooleanSearchParam } from '@inngest/components/hooks/useSearchParam';
import { IconFunction } from '@inngest/components/icons/Function';

import { useEnvironment } from '@/components/Environments/environment-context';
import Header, { type HeaderLink } from '@/components/Header/old/Header';

export const FunctionsHeader = () => {
  const env = useEnvironment();
  const [archived] = useBooleanSearchParam('archived');

  const navLinks: HeaderLink[] = [
    {
      active: !archived,
      href: `/env/${env.slug}/functions`,
      text: 'Active',
    },
    {
      active: archived,
      href: `/env/${env.slug}/functions?archived=true`,
      text: 'Archived',
    },
  ];

  return (
    <Header
      icon={<IconFunction className="h-5 w-5 text-white" />}
      links={navLinks}
      title="Functions"
    />
  );
};
