import LoadingIcon from '@/components/Icons/LoadingIcon';
import SplitView from '@/components/SignIn/SplitView';
import { useClerk, useSignIn } from '@clerk/tanstack-react-start';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect, useState } from 'react';

type AgentDeepLinkSearch = {
  organization_id?: string;
  redirect_url?: string;
  ticket?: string;
};

//
// Clerk tickets are single-use. Track consumed tickets at module scope
// so React Strict Mode's development double-mount doesn't attempt to
// consume the same ticket twice.
const consumedTickets = new Set<string>();

export const Route = createFileRoute('/(auth)/agent-deep-link')({
  component: AgentDeepLink,
  validateSearch: (search: Record<string, unknown>): AgentDeepLinkSearch => ({
    organization_id:
      typeof search.organization_id === 'string'
        ? search.organization_id
        : undefined,
    redirect_url:
      typeof search.redirect_url === 'string' &&
      search.redirect_url.startsWith('/')
        ? search.redirect_url
        : undefined,
    ticket: typeof search.ticket === 'string' ? search.ticket : undefined,
  }),
});

function AgentDeepLink() {
  const { organization_id, redirect_url, ticket } = Route.useSearch();
  const { isLoaded, signIn } = useSignIn();
  const { setActive, signOut } = useClerk();
  const navigate = useNavigate();
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (!isLoaded || !ticket || !redirect_url) return;
    if (consumedTickets.has(ticket)) return;
    consumedTickets.add(ticket);

    const activate = async () => {
      //
      // Clear any existing session so the ticket sign-in doesn't compete
      // with background session-refresh requests for Clerk rate-limit budget.
      await signOut().catch(() => {});

      const { createdSessionId } = await signIn.create({
        strategy: 'ticket',
        ticket,
      });

      await setActive({
        session: createdSessionId,
        ...(organization_id && { organization: organization_id }),
      });

      //
      // Client-side navigate to preserve Clerk's in-memory session state.
      navigate({ to: redirect_url, replace: true });
    };

    activate().catch((err) => {
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
