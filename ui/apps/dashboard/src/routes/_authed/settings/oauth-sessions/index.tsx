import { useAuth } from '@clerk/tanstack-react-start';
import { createFileRoute } from '@tanstack/react-router';

import { OAuthSessions } from '@/components/OAuth/OAuthSessions';

export const Route = createFileRoute('/_authed/settings/oauth-sessions/')({
  component: OAuthSessionsPage,
});

function OAuthSessionsPage() {
  const { userId, orgId } = useAuth();
  return (
    <div className="mx-auto flex w-full max-w-[900px] flex-col gap-12 px-4 py-12">
      <OAuthSessions key={JSON.stringify([userId, orgId])} />
    </div>
  );
}
