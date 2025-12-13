import { SignOutButton } from '@/components/Auth/SignOutButton';
import SplitView from '@/components/SignIn/SplitView';

import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/(auth)/sign-out')({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <SplitView>
      <div className="flex flex-row items-center justify-center">
        <SignOutButton />
      </div>
    </SplitView>
  );
}
