import { Marketplace } from '@/gql/graphql';

type APIKeyPermissions = {
  marketplace: Marketplace | null;
  organizationRole?: string;
};

export const canManageAPIKeys = ({
  marketplace,
  organizationRole,
}: APIKeyPermissions) => {
  // Vercel and DigitalOcean sessions use their provisioned admin over JWT.
  return (
    marketplace === Marketplace.Vercel ||
    marketplace === Marketplace.DigitalOcean ||
    organizationRole === 'org:admin'
  );
};
