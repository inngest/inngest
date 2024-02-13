import AppNavigation from '@/components/Navigation/AppNavigation';
import CreateEnvironment from './CreateEnvironment';

export default function Create() {
  return (
    <div className="flex h-full flex-col">
      <AppNavigation environmentSlug="all" />
      <div className="mx-auto w-full max-w-[860px] px-12 py-16">
        <CreateEnvironment />
      </div>
    </div>
  );
}
