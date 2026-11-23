import { ConnectedVerification } from '@/components/ComputeVerification/ConnectedVerification';
import { useAuth, useClerk } from '@clerk/tanstack-react-start';
import { createFileRoute, getRouteApi } from '@tanstack/react-router';
import { VerificationLayout } from '@/components/ComputeVerification/ComputeVerification';
import type { EntrySource } from '@/components/ComputeVerification/types';
import TopBar from '@/components/Layout/TopBar';
import BottomBar from '@/components/Layout/BottomBar';
import { Panel, Heading } from '@/components/ComputeVerification/shared';

export const Route = createFileRoute('/_authed/verify')({
  validateSearch: (search: Record<string, unknown>) => ({
    account_id:
      typeof search.account_id === 'string'
        ? search.account_id.toLowerCase()
        : '',
    source: (search.source === 'dashboard' || search.source === 'cli'
      ? search.source
      : 'api') as EntrySource,
    return_to:
      typeof search.return_to === 'string' &&
      /^\/env\/[^/?#]+\/sandboxes\/?$/.test(search.return_to)
        ? search.return_to
        : undefined,
  }),
  component: VerificationRoute,
});
const authed = getRouteApi('/_authed');
function VerificationRoute(): JSX.Element {
  const { userId, orgId } = useAuth();
  const { profile } = authed.useLoaderData();
  const clerk = useClerk();
  const search = Route.useSearch();
  if (
    !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(
      search.account_id,
    )
  ) {
    return (
      <VerificationLayout
        header={
          <TopBar
            focused
            profile={profile}
            showOnboardingWidget={() => {
              window.location.assign('/env/production');
            }}
          />
        }
        footer={<BottomBar focused />}
      >
        <Panel>
          <Heading title="Invalid verification link">
            Open the verification link from your original request.
          </Heading>
        </Panel>
      </VerificationLayout>
    );
  }
  return (
    <ConnectedVerification
      key={`${userId}:${orgId}:${search.account_id}`}
      profile={profile}
      profileID={userId ?? 'marketplace'}
      onManageEmail={() => clerk.openUserProfile()}
      {...search}
    />
  );
}
