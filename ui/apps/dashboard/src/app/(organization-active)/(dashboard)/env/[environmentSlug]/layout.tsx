import type { ReactNode } from 'react';
import { Alert } from '@inngest/components/Alert/Alert';

import { ArchivedEnvBanner } from '@/components/ArchivedEnvBanner';
import { BillingBanner } from '@/components/BillingBanner';
import { getEnv } from '@/components/Environments/data';
import { EnvironmentProvider } from '@/components/Environments/environment-context';
import Layout from '@/components/Layout/Layout';
import Toaster from '@/components/Toaster';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import type { Environment } from '@/utils/environments';

type RootLayoutProps = {
  params: {
    environmentSlug: string;
  };
  children: ReactNode;
};

const NotFound = () => (
  <div className="mt-16 flex place-content-center">
    <Alert severity="warning">Environment not found.</Alert>
  </div>
);

const Env = ({ env, children }: { env?: Environment; children: ReactNode }) =>
  env ? (
    <>
      <ArchivedEnvBanner env={env} />
      <EnvironmentProvider env={env}>{children}</EnvironmentProvider>
    </>
  ) : (
    <NotFound />
  );

export default async function RootLayout({
  params: { environmentSlug },
  children,
}: RootLayoutProps) {
  const env = await getEnv(environmentSlug);

  let entitlementUsage;
  try {
    entitlementUsage = (await graphqlAPI.request(entitlementUsageQuery)).account.entitlementUsage;
  } catch (e) {
    console.error(e);
    return null;
  }

  return (
    <>
      <Layout activeEnv={env}>
        {entitlementUsage && <BillingBanner entitlementUsage={entitlementUsage} />}
        <Env env={env}>{children}</Env>
      </Layout>
      <Toaster
        toastOptions={{
          // Ensure that the toast is clickable when there are overlays/modals
          className: 'pointer-events-auto',
        }}
      />
    </>
  );
}

const entitlementUsageQuery = graphql(`
  query EntitlementUsage {
    account {
      id
      entitlementUsage {
        runCount {
          current
          limit
        }
      }
    }
  }
`);
