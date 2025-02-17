'use client';

import { usePathname } from 'next/navigation';
import { Header } from '@inngest/components/Header/Header';

import CreateKeyButton from '../../app/(organization-active)/(dashboard)/env/[environmentSlug]/manage/[ingestKeys]/CreateKeyButton';
import { EventKeyInfo } from './EventKeyInfo';
import { SigningKeyInfo } from './SigningKeyInfo';
import { WebhookInfo } from './WebhookInfo';

export const ManageHeader = () => {
  const pathname = usePathname();

  return (
    <Header
      breadcrumb={[
        ...(pathname.includes('/webhooks') ? [{ text: 'Webhooks' }] : []),
        ...(pathname.includes('/keys') ? [{ text: 'Event Keys' }] : []),
        ...(pathname.includes('/signing-key') ? [{ text: 'Signing Key' }] : []),
      ]}
      infoIcon={
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
