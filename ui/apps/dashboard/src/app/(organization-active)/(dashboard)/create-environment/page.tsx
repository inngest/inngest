import AppNavigation from '@/components/Navigation/old/AppNavigation';
import getAllEnvironments from '@/queries/server-only/getAllEnvironments';
import CreateEnvironment from './CreateEnvironment';

export default async function Create() {
  const envs = await getAllEnvironments();
  return (
    <div className="flex h-full flex-col">
      <AppNavigation envs={envs} envSlug="all" />
      <div className="mx-auto w-full max-w-[860px] px-12 py-16">
        <CreateEnvironment />
      </div>
    </div>
  );
}
