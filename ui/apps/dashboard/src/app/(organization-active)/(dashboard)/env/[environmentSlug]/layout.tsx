import { ArchivedEnvBanner } from '@/components/ArchivedEnvBanner';
import { getEnvs } from '@/components/Environments/data';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import AppNavigation from '@/components/Navigation/old/AppNavigation';
import Toaster from '@/components/Toaster';
import { EnvironmentProvider } from '../../../../../components/Environments/environment-context';

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
  const { env, envs } = await getEnvs(environmentSlug);
  const newIANav = await getBooleanFlag('new-ia-nav');

  return (
    <>
      <div className="isolate flex h-full flex-col">
        {newIANav ? (
          <div className="p-4">coming soon...</div>
        ) : (
          <AppNavigation envs={envs} activeEnv={env} envSlug={environmentSlug} />
        )}
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
