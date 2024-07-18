import Envs from '@/components/Environments/Environments';
import AppNavigation from '@/components/Navigation/old/AppNavigation';
import Toaster from '@/components/Toaster';

export default async function EnvsPage() {
  return (
    <>
      <div className="flex h-full flex-col">
        <AppNavigation envSlug="all" />
        <Envs />
      </div>
      <Toaster />
    </>
  );
}
