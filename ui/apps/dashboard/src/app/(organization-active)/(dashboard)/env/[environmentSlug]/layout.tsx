import type { ReactNode } from 'react';
import { Alert } from '@inngest/components/Alert/Alert';

import { SharedContextProvider } from '@/app/SharedContextProvider';
import { ArchivedEnvBanner } from '@/components/ArchivedEnvBanner';
import { getEnv } from '@/components/Environments/data';
import { EnvironmentProvider } from '@/components/Environments/environment-context';
import Layout from '@/components/Layout/Layout';
import Toaster from '@/components/Toaster';
import type { Environment } from '@/utils/environments';

type RootLayoutProps = {
  params: Promise<{
    environmentSlug: string;
  }>;
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

export default async function RootLayout(props: RootLayoutProps) {
  const params = await props.params;

  const { environmentSlug } = params;

  const { children } = props;

  const env = await getEnv(environmentSlug);

  return (
    <>
      <Layout activeEnv={env}>
        <Env env={env}>
          <SharedContextProvider>{children}</SharedContextProvider>
        </Env>
      </Layout>
      <Toaster />
    </>
  );
}
