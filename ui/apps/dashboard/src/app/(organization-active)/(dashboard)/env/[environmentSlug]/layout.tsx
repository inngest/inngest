import type { ReactNode } from 'react';
import dynamic from 'next/dynamic';
import { Alert } from '@inngest/components/Alert/Alert';

import { SharedContextProvider } from '@/app/SharedContextProvider';
import { ArchivedEnvBanner } from '@/components/ArchivedEnvBanner';
import { getEnv } from '@/components/Environments/data';
import { EnvironmentProvider } from '@/components/Environments/environment-context';
import Layout from '@/components/Layout/Layout';
import Toaster from '@/components/Toaster';
import type { Environment } from '@/utils/environments';

const Feedback = dynamic(() => import('@/components/Surveys/Feedback'), {
  ssr: false,
});

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

  return (
    <>
      <Layout activeEnv={env}>
        <Env env={env}>
          <SharedContextProvider>{children}</SharedContextProvider>
          <Feedback />
        </Env>
      </Layout>
      <Toaster />
    </>
  );
}
