import { Header } from '@inngest/components/Header/Header';

import { FunctionInfo } from '@/components/Functions/FunctionInfo';
import { FunctionList } from '@/components/Functions/FunctionsList';

type FunctionLayoutProps = {
  params: {
    environmentSlug: string;
  };
  searchParams: {
    archived?: string;
  };
};

export default async function FunctionPage({
  params: { environmentSlug },
  searchParams: { archived: archivedParam },
}: FunctionLayoutProps) {
  const archived = archivedParam === 'true';

  return (
    <>
      <Header breadcrumb={[{ text: 'Functions' }]} infoIcon={<FunctionInfo />} />
      <div className="bg-canvasBase no-scrollbar flex w-full flex-col">
        <FunctionList envSlug={environmentSlug} archived={archived} />
      </div>
    </>
  );
}
