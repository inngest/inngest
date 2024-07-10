import { getEnvs } from '@/components/Environments/data';
import AppNavigation from '@/components/Navigation/old/AppNavigation';
import Toaster from '@/components/Toaster';
import { ArchivedEnvBanner } from '../Environments/ArchivedEnvBanner';
import { EnvironmentProvider } from '../Environments/EnvContext';

type RootLayoutProps = {
  environmentSlug: string;
  children: React.ReactNode;
};

export default async function RootLayout({ environmentSlug, children }: RootLayoutProps) {
  const { env, envs } = await getEnvs(environmentSlug);
  return (
    <>
      <div className="isolate flex h-full flex-col">
        <AppNavigation envs={envs} activeEnv={env} />
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
