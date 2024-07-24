import { ArchivedEnvBanner } from '@/components/ArchivedEnvBanner';
import { getEnv } from '@/components/Environments/data';
import { EnvironmentProvider } from '@/components/Environments/environment-context';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Layout from '@/components/Layout/Layout';
import AppNavigation from '@/components/Navigation/old/AppNavigation';
import Toaster from '@/components/Toaster';

type RootLayoutProps = {
  params: {
    environmentSlug: string;
  };
  children: React.ReactNode;
};

export default async function RootLayout({
  params: { environmentSlug },
  children,
}: RootLayoutProps) {
  const env = await getEnv(environmentSlug);
  const newIANav = await getBooleanFlag('new-ia-nav');

  return newIANav ? (
    <Layout activeEnv={env}>
      <ArchivedEnvBanner env={env} />
      <EnvironmentProvider env={env}>{children}</EnvironmentProvider>
    </Layout>
  ) : (
    <>
      <div className="isolate flex h-full flex-col">
        <AppNavigation activeEnv={env} envSlug={environmentSlug} />

        <ArchivedEnvBanner env={env} />
        <EnvironmentProvider env={env}>{children}</EnvironmentProvider>
      </div>
      <Toaster
        toastOptions={{
          // Ensure that the toast is clickable when there are overlays/modals
          className: 'pointer-events-auto',
        }}
      />
    </>
  );
}
