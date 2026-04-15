import {
  ClientOnly,
  createFileRoute,
  useNavigate,
} from '@tanstack/react-router';

import ExperimentDetailPage from '@/components/Experiments/ExperimentDetailPage';

type ExperimentDetailSearch = {
  field?: string[];
};

export const Route = createFileRoute(
  '/_authed/env/$envSlug/experiments/$experimentName/',
)({
  component: ExperimentDetailComponent,
  validateSearch: (search: Record<string, unknown>): ExperimentDetailSearch => {
    const field = search.field;

    return {
      field: Array.isArray(field)
        ? field.filter((value): value is string => typeof value === 'string')
        : typeof field === 'string'
        ? [field]
        : undefined,
    };
  },
});

function ExperimentDetailComponent() {
  const navigate = useNavigate();
  const { envSlug, experimentName } = Route.useParams();
  const search = Route.useSearch();

  return (
    <ClientOnly>
      <ExperimentDetailPage
        environmentSlug={envSlug}
        experimentName={decodeURIComponent(experimentName)}
        selectedFieldKeys={search.field ?? []}
        onSelectedFieldKeysChange={(fields) => {
          void navigate({
            to: Route.to,
            params: { envSlug, experimentName },
            search: fields.length > 0 ? { field: fields } : {},
            replace: true,
          });
        }}
      />
    </ClientOnly>
  );
}
