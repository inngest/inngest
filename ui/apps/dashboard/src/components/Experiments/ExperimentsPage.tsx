import { Header } from '@inngest/components/Header/Header';
import { ExperimentsTable } from '@inngest/components/Experiments';
import { useNavigate } from '@tanstack/react-router';

import { pathCreator } from '@/utils/urls';

import { useExperimentsList } from './useExperiments';

export default function ExperimentsPage({
  environmentSlug,
}: {
  environmentSlug: string;
}) {
  const navigate = useNavigate();
  const { data, isPending, error, refetch } = useExperimentsList();

  return (
    <>
      <Header breadcrumb={[{ text: 'Experiments' }]} />
      <div className="flex flex-1 flex-col overflow-hidden p-3">
        <ExperimentsTable
          data={data}
          isPending={isPending}
          error={error ?? null}
          refetch={() => {
            void refetch();
          }}
          getRowHref={(item) =>
            pathCreator.experiment({
              envSlug: environmentSlug,
              experimentName: item.experimentName,
            })
          }
          onRowClick={(item) => {
            void navigate({
              to: pathCreator.experiment({
                envSlug: environmentSlug,
                experimentName: item.experimentName,
              }),
            });
          }}
        />
      </div>
    </>
  );
}
