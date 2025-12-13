import SplitView from '@/components/SignIn/SplitView';
import { SignOutButton } from '@clerk/tanstack-react-start';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

export const Route = createFileRoute('/(auth)/sign-out')({
  component: RouteComponent,
});

function RouteComponent() {
  const navigate = useNavigate();

  return (
    <SplitView>
      <div className="flex flex-row items-center justify-center">
        <SignOutButton
          signOutCallback={() => {
            //
            // Ensure proper cleanup by navigating to sign-in after sign-out completes
            navigate({ to: '/sign-in', replace: true });
          }}
        />
      </div>
    </SplitView>
  );
}
