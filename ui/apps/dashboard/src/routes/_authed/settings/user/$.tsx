import { UserProfile } from '@clerk/tanstack-react-start';
import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_authed/settings/user/$')({
  gcTime: 0,
  ssr: false,
  component: UserSettingsPage,
  beforeLoad: ({ location }) => {
    if (
      location.pathname === '/settings/user' ||
      location.pathname === '/settings/user/'
    ) {
      throw redirect({ to: '/settings/user/$', params: { _splat: 'profile' } });
    }
  },
});

function UserSettingsPage() {
  const { _splat } = Route.useParams();

  return (
    <div className="flex flex-col justify-start">
      {_splat === 'profile' && (
        <UserProfile
          routing="path"
          path="/settings/user/profile"
          appearance={{
            layout: {
              logoPlacement: 'none',
            },
            elements: {
              navbar: 'hidden',
              scrollBox: 'bg-canvasBase shadow-none',
              pageScrollBox: 'pt-6 px-2',
            },
          }}
        >
          <UserProfile.Page label="security" />
        </UserProfile>
      )}
      {_splat === 'security' && (
        <UserProfile
          routing="path"
          path="/settings/user/security"
          appearance={{
            layout: {
              logoPlacement: 'none',
            },
            elements: {
              navbar: 'hidden',
              scrollBox: 'bg-canvasBase shadow-none',
              pageScrollBox: 'pt-0 px-2',
            },
          }}
        >
          <UserProfile.Page label="account" />
        </UserProfile>
      )}
    </div>
  );
}
