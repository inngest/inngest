import LoadingIcon from '@/components/Icons/LoadingIcon';
import SplitView from '@/components/SignIn/SplitView';
import {
  stripDeepLinkParams,
  validateAgentDeepLinkSearch,
} from '@/lib/deepLinkUtils';
import { useClerk, useSignIn } from '@clerk/tanstack-react-start';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect, useRef, useState } from 'react';

//
// Clerk sign-in tokens are single-use and some clerk operations below can cause
// re-mounts, so we track consumed tickets to prevent double-consumption.
const consumedTickets = new Set<string>();

export const Route = createFileRoute('/(auth)/agent-deep-link')({
  component: AgentDeepLink,
  validateSearch: validateAgentDeepLinkSearch,
});

function AgentDeepLink() {
  const { organization_id, redirect_url, ticket } = Route.useSearch();
  const { isLoaded, signIn } = useSignIn();
  const { setActive, signOut } = useClerk();
  const navigate = useNavigate();
  const isActivatingRef = useRef(false);
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (!isLoaded || !ticket || !redirect_url) {
      return;
    }

    if (isActivatingRef.current) {
      return;
    }

    if (consumedTickets.has(ticket)) {
      return;
    }

    consumedTickets.add(ticket);

    isActivatingRef.current = true;

    const activate = async () => {
      //
      // Clear any existing session so the ticket sign-in doesn't compete
      // with background session-refresh and hit clerk rate limits.
      await signOut().catch(() => {});

      const { createdSessionId } = await signIn.create({
        strategy: 'ticket',
        ticket,
      });

      await setActive({
        session: createdSessionId,
        ...(organization_id && { organization: organization_id }),
      });

      await navigate({
        //
        // Deep-link auth params are only for the sign-in handoff.
        // Strip them before navigating so TanStack Router doesn't
        // re-serialize parseable values like Unix timestamps.
        href: stripDeepLinkParams(redirect_url),
        replace: true,
      });
    };

    activate().catch((err) => {
      isActivatingRef.current = false;
      console.error('Agent deep link sign-in failed:', err);
      setError('This deep link is invalid or has expired.');
    });
  }, [
    isLoaded,
    ticket,
    redirect_url,
    organization_id,
    signIn,
    setActive,
    signOut,
  ]);

  if (!ticket || !redirect_url) {
    return (
      <SplitView>
        <div className="mx-auto my-auto text-center">Invalid deep link.</div>
      </SplitView>
    );
  }

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        {error ? (
          <span>{error}</span>
        ) : (
          <div className="flex items-center justify-center gap-2">
            <LoadingIcon />
            Signing you in...
          </div>
        )}
      </div>
    </SplitView>
  );
}
