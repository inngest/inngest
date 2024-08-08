import { RiToolsLine } from '@remixicon/react';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import OldHeader from '@/components/Header/old/Header';
import { ManageHeader } from '@/components/Manage/Header';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import ChildEmptyState from './ChildEmptyState';
import CreateKeyButton from './[ingestKeys]/CreateKeyButton';

type ManageLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
  };
};

export default async function ManageLayout({ children, params }: ManageLayoutProps) {
  const environment = await getEnvironment({
    environmentSlug: params.environmentSlug,
  });
  const newIANav = await getBooleanFlag('new-ia-nav');

  const isChildEnvironment = environment.hasParent;
  const keysPath = `/env/${params.environmentSlug}/manage/keys`;
  const hooksPath = `/env/${params.environmentSlug}/manage/webhooks`;
  const signingPath = `/env/${params.environmentSlug}/manage/signing-key`;

  if (isChildEnvironment) {
    return <ChildEmptyState />;
  }

  return (
    <>
      {newIANav ? (
        <ManageHeader />
      ) : (
        <OldHeader
          title="Manage Environment"
          icon={<RiToolsLine className="h-4 w-4 text-white" />}
          links={[
            {
              href: keysPath,
              text: 'Event Keys',
            },
            {
              href: hooksPath,
              text: 'Webhooks',
            },
            {
              href: signingPath,
              text: 'Signing Key',
            },
          ]}
          action={<CreateKeyButton />}
        />
      )}
      {children}
    </>
  );
}
