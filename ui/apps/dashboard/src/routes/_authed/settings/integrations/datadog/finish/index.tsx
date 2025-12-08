import { createFileRoute } from '@tanstack/react-router';

import DatadogSetupPage from '@/components/Settings/DatadogSetupPage';
import FinishPage from '@/components/DatadogIntegration/FinishPage';
import { MetricsEntitlements } from '@/queries/server/entitlements';

type DatadogFinishSearch = {
  site?: string;
  domain?: string;
  code?: string;
  dd_oid?: string;
  dd_org_name?: string;
};

export const Route = createFileRoute(
  '/_authed/settings/integrations/datadog/finish/',
)({
  component: DatadogFinishPage,
  validateSearch: (search: Record<string, unknown>): DatadogFinishSearch => ({
    site: search.site as string | undefined,
    domain: search.domain as string | undefined,
    code: search.code as string | undefined,
    dd_oid: search.dd_oid as string | undefined,
    dd_org_name: search.dd_org_name as string | undefined,
  }),
  loader: async () => {
    const metricsEntitlements = await MetricsEntitlements();
    return { metricsEntitlements };
  },
});

function DatadogFinishPage() {
  const { metricsEntitlements } = Route.useLoaderData();
  const { site, domain, code, dd_oid, dd_org_name } = Route.useSearch();

  return (
    <DatadogSetupPage
      title="Connect to Datadog"
      showSubtitleDocsLink={false}
      content={
        <FinishPage
          site={site}
          domain={domain}
          code={code}
          dd_oid={dd_oid}
          dd_org_name={dd_org_name}
        />
      }
      metricsEntitlements={metricsEntitlements}
    />
  );
}
