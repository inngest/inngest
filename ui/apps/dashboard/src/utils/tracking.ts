import { useOrganization, useUser } from '@clerk/nextjs';

type TrackingUser = {
  external_id: string | null;
  email?: string;
  name: string | null;
  account_id?: unknown;
};

type BaseEventData = {
  v: string;
  user?: TrackingUser;
};

type TrackingEvent<T = Record<string, any>> = BaseEventData & {
  name: string;
  data: T;
};

export function useTrackingUser() {
  const { user } = useUser();
  const { organization } = useOrganization();

  if (!user || !organization) return undefined;

  return {
    external_id: user.externalId,
    email: user.primaryEmailAddress?.emailAddress,
    name: user.fullName,
    account_id: organization.publicMetadata.accountID,
  };
}

export function trackEvent({ name, data, user, v }: TrackingEvent): void {
  if (typeof window === 'undefined') return;

  const event = {
    name,
    data,
    user,
    v,
  };

  window.inngest.send(event);
}
