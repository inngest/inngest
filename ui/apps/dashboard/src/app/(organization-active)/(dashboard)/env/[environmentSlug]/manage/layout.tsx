import { WrenchIcon } from '@heroicons/react/20/solid';

import Header from '@/components/Header/Header';
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

  const isChildEnvironment = environment.hasParent;

  const navLinks = [
    {
      href: `/env/${params.environmentSlug}/manage/keys`,
      text: 'Event Keys',
    },
    {
      href: `/env/${params.environmentSlug}/manage/webhooks`,
      text: 'Webhooks',
    },
    {
      href: `/env/${params.environmentSlug}/manage/signing-key`,
      text: 'Signing Key',
    },
  ];

  return (
    <>
      {isChildEnvironment ? (
        <ChildEmptyState />
      ) : (
        <>
          <Header
            title="Manage Environment"
            icon={<WrenchIcon className="h-4 w-4 text-white" />}
            links={navLinks}
            action={!environment.isArchived && <CreateKeyButton />}
          />
          {children}
        </>
      )}
    </>
  );
}
