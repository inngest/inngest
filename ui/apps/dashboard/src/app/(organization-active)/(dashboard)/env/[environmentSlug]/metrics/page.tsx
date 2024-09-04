'use client';

import { NewButton } from '@inngest/components/Button';
import { Header } from '@inngest/components/Header/Header';
import { MetricsActionMenu } from '@inngest/components/Metrics/ActionMenu';
import { Dashboard } from '@inngest/components/Metrics/Dashboard';
import { subtractDuration } from '@inngest/components/utils/date';
import { RiRefreshLine } from '@remixicon/react';
import { useQuery } from 'urql';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { GetBillingPlanDocument } from '@/gql/graphql';

export const AppFilterDocument = graphql(`
  query AppFilter($envSlug: String!) {
    env: envBySlug(slug: $envSlug) {
      apps {
        externalID
        id
        name
      }
    }
  }
`);

const FunctionFilterDocument = graphql(`
  query FunctionFilter($environmentID: ID!, $archived: Boolean) {
    workspace(id: $environmentID) {
      workflows(archived: $archived) {
        data {
          id
          name
          isArchived
        }
      }
    }
  }
`);

type MetricsProps = {
  params: {
    environmentSlug: string;
  };
};

export default function MetricsPage({ params: { environmentSlug: envSlug } }: MetricsProps) {
  const env = useEnvironment();

  const [{ data: planData }] = useQuery({
    query: GetBillingPlanDocument,
  });

  const logRetention = Number(planData?.account.plan?.features.log_retention);
  const upgradeCutoff = subtractDuration(new Date(), { days: logRetention || 7 });

  const [{ data: appData }] = useQuery({
    query: AppFilterDocument,
    variables: { envSlug },
  });

  const [{ data: functionsData }] = useQuery({
    query: FunctionFilterDocument,
    variables: {
      archived: false,
      environmentID: env.id,
    },
  });

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Metrics' }]}
        action={
          <div className="flex flex-row items-center justify-end gap-x-1">
            <NewButton
              kind="primary"
              appearance="outlined"
              label="Refresh page"
              icon={<RiRefreshLine />}
              iconSide="left"
            />
            <MetricsActionMenu
              autoRefresh={false}
              setAutoRefresh={() => null}
              intervalSeconds={3}
            />
          </div>
        }
      />
      <div className="bg-canvasSubtle mx-auto flex h-full w-full flex-col">
        <Dashboard
          apps={appData?.env?.apps.map((app) => ({
            id: app.id,
            name: app.externalID,
          }))}
          functions={functionsData?.workspace.workflows.data}
          upgradeCutoff={upgradeCutoff}
        />
      </div>
    </>
  );
}
