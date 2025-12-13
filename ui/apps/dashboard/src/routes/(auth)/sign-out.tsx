import { SignOutButton } from '@/components/Auth/SignOutButton';
import SplitView from '@/components/SignIn/SplitView';
import { useAuth } from '@clerk/tanstack-react-start';

import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';

export const Route = createFileRoute('/(auth)/sign-out')({
  component: RouteComponent,
});

function RouteComponent() {
  const { isSignedIn } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isSignedIn) {
      navigate({ to: '/sign-in/$' });
    }
  }, [isSignedIn, navigate]);

  return (
    <SplitView>
      <div className="flex flex-row items-center justify-center">
        <SignOutButton />
      </div>
    </SplitView>
  );
}
