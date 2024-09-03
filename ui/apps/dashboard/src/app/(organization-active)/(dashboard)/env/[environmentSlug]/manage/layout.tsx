import { ManageHeader } from '@/components/Manage/Header';
import { getEnvironment } from '@/queries/server-only/getEnvironment';
import ChildEmptyState from './ChildEmptyState';

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

  if (isChildEnvironment) {
    return <ChildEmptyState />;
  }

  return (
    <>
      <ManageHeader />
      {children}
    </>
  );
}
