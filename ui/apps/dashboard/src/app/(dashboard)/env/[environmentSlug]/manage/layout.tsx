import { Cog6ToothIcon, KeyIcon, WrenchIcon } from '@heroicons/react/20/solid';

import Header from '@/components/Header/Header';
import WebhookIcon from '@/icons/webhookIcon.svg';
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
      icon: <KeyIcon className="w-3.5" />,
    },
    {
      href: `/env/${params.environmentSlug}/manage/webhooks`,
      text: 'Webhooks',
      icon: <WebhookIcon className="w-3.5" />,
    },
    {
      href: `/env/${params.environmentSlug}/manage/signing-key`,
      text: 'Signing Key',
      icon: <KeyIcon className="w-3.5" />,
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
            action={<CreateKeyButton environmentSlug={params.environmentSlug} />}
          />
          {children}
        </>
      )}
    </>
  );
}
