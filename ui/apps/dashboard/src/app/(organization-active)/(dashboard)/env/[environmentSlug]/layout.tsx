import type { ReactNode } from 'react';
import { Alert } from '@inngest/components/Alert/Alert';

import { ArchivedEnvBanner } from '@/components/ArchivedEnvBanner';
import { getEnv } from '@/components/Environments/data';
import { EnvironmentProvider } from '@/components/Environments/environment-context';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Layout from '@/components/Layout/Layout';
import AppNavigation from '@/components/Navigation/old/AppNavigation';
import Toaster from '@/components/Toaster';
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
  const newIANav = await getBooleanFlag('new-ia-nav');

  return (
    <>
      {newIANav ? (
        <Layout activeEnv={env}>
          <Env env={env}>{children}</Env>
        </Layout>
      ) : (
        <div className="isolate flex h-full flex-col">
          <AppNavigation activeEnv={env} envSlug={environmentSlug} />
          <Env env={env}>{children}</Env>
        </div>
      )}
      <Toaster
        toastOptions={{
          // Ensure that the toast is clickable when there are overlays/modals
          className: 'pointer-events-auto',
        }}
      />
    </>
  );
}
