'use client';

import {
  ArchiveBoxIcon,
  ChartBarSquareIcon,
  CodeBracketSquareIcon,
  CommandLineIcon,
  FolderIcon,
} from '@heroicons/react/20/solid';

import Header, { type HeaderLink } from '@/components/Header/Header';
import { Tag } from '@/components/Tag/Tag';
import { useFunction } from '@/queries';
import ArchiveFunctionButton from './ArchiveButton';
import PauseFunctionButton from './PauseButton';

type FunctionLayoutProps = {
  children: React.ReactNode;
  params: {
    environmentSlug: string;
    slug: string;
  };
};

export default function FunctionLayout({ children, params }: FunctionLayoutProps) {
  const functionSlug = decodeURIComponent(params.slug);
  const [{ data, fetching }] = useFunction({
    environmentSlug: params.environmentSlug,
    functionSlug,
  });

  const fn = data?.workspace.workflow;
  const { isArchived = false } = fn ?? {};
  const isPaused = !isArchived && !data?.workspace.workflow?.current;

  const emptyData = !data || fetching || !fn;
  const navLinks: HeaderLink[] = [
    {
      href: `/env/${params.environmentSlug}/functions/${params.slug}`,
      text: 'Dashboard',
      icon: <ChartBarSquareIcon className="w-3.5" />,
      active: 'exact',
    },
    {
      href: `/env/${params.environmentSlug}/functions/${params.slug}/logs`,
      text: 'Logs',
      icon: <CommandLineIcon className="w-3.5" />,
    },
  ];
  return (
    <>
      <Header
        icon={<CodeBracketSquareIcon className="h-3.5 w-3.5 text-white" />}
        title={!data || fetching ? '...' : fn?.name || functionSlug}
        links={navLinks}
        action={
          !emptyData && (
            <div className="flex items-center gap-2">
              {/* Disable buttons that do not yet work */}
              <div className="flex items-center gap-2 pr-2">
                <PauseFunctionButton
                  environmentSlug={params.environmentSlug}
                  functionSlug={functionSlug}
                  disabled={isArchived}
                />
                <ArchiveFunctionButton
                  environmentSlug={params.environmentSlug}
                  functionSlug={functionSlug}
                />
              </div>
              {/* <Button context="dark">
              <RocketLaunchIcon className="h-3" />
              Run Function
            </Button> */}
            </div>
          )
        }
        tag={
          !emptyData && isArchived ? (
            <Tag size="sm">Archived</Tag>
          ) : !emptyData && isPaused ? (
            <Tag size="sm" className="text-amber-500">
              Paused
            </Tag>
          ) : null
        }
      />
      {children}
    </>
  );
}
