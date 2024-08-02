'use client';

import { usePathname } from 'next/navigation';

import { Header } from '@/components/Header/Header';
import CreateKeyButton from '../../app/(organization-active)/(dashboard)/env/[environmentSlug]/manage/[ingestKeys]/CreateKeyButton';
import { EventKeyInfo } from './EventKeyInfo';
import { SigningKeyInfo } from './SigningKeyInfo';
import { WebhookInfo } from './WebhookInfo';

export const ManageHeader = ({ envSlug }: { envSlug: string }) => {
  const keysPath = `/env/${envSlug}/manage/keys`;
  const hooksPath = `/env/${envSlug}/manage/webhooks`;
  const signingPath = `/env/${envSlug}/manage/signing-key`;
  const pathname = usePathname();

  return (
    <Header
      breadcrumb={[
        ...(pathname.includes('/webhooks') ? [{ text: 'Webhooks', href: hooksPath }] : []),
        ...(pathname.includes('/keys') ? [{ text: 'Event Keys', href: keysPath }] : []),
        ...(pathname.includes('/signing-key') ? [{ text: 'Signing Key', href: signingPath }] : []),
      ]}
      icon={
        <>
          {pathname.includes('/webhooks') && <WebhookInfo />}
          {pathname.includes('/keys') && <EventKeyInfo />}
          {pathname.includes('/signing-key') && <SigningKeyInfo />}
        </>
      }
      action={<CreateKeyButton />}
    />
  );
};
