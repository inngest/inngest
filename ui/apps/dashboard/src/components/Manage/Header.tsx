'use client';

import { usePathname } from 'next/navigation';

import { Header } from '@/components/Header/Header';
import CreateKeyButton from '../../app/(organization-active)/(dashboard)/env/[environmentSlug]/manage/[ingestKeys]/CreateKeyButton';
import { ManageInfo } from './ManageInfo';

export const ManageHeader = ({ envSlug }: { envSlug: string }) => {
  const managePath = `/env/${envSlug}/manage`;
  const keysPath = `/env/${envSlug}/manage/keys`;
  const hooksPath = `/env/${envSlug}/manage/webhooks`;
  const signingPath = `/env/${envSlug}/manage/signing-key`;
  const pathname = usePathname();

  return (
    <Header
      breadcrumb={[
        { text: 'Manage Environment', href: managePath },

        ...(pathname?.includes('/keys') ? [{ text: 'Keys', href: keysPath }] : []),
        ...(pathname?.includes('/webhooks') ? [{ text: 'Webhooks', href: hooksPath }] : []),
        ...(pathname?.includes('/signing-key') ? [{ text: 'Signing Key', href: signingPath }] : []),
      ]}
      icon={<ManageInfo />}
      tabs={[
        {
          href: keysPath,
          children: 'Event Keys',
        },
        {
          href: hooksPath,
          children: 'Webhooks',
        },
        {
          href: signingPath,
          children: 'Signing Key',
        },
      ]}
      action={<CreateKeyButton />}
    />
  );
};
