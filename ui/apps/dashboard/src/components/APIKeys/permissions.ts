type APIKeyPermissions = {
  isMarketplace: boolean;
  organizationRole?: string;
};

export const canManageAPIKeys = ({
  isMarketplace,
  organizationRole,
}: APIKeyPermissions) => {
  // Marketplace sessions use JWT auth and have no Clerk organization role.
  return isMarketplace || organizationRole === 'org:admin';
};
