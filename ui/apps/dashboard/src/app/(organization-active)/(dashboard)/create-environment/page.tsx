import AppNavigation from '@/components/Navigation/old/AppNavigation';
import CreateEnvironment from './CreateEnvironment';

export default async function Create() {
  return (
    <div className="flex h-full flex-col">
      <AppNavigation envSlug="all" />
      <div className="mx-auto w-full max-w-[860px] px-12 py-16">
        <CreateEnvironment />
      </div>
    </div>
  );
}
