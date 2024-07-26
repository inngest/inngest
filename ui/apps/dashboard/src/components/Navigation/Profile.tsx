import { getProfile } from '@/queries/server-only/profile';

export const Profile = async ({ collapsed }: { collapsed: boolean }) => {
  const { user, org } = await getProfile();

  return (
    <div className="border-subtle flex h-16 flex-row items-center justify-start border-t px-4">
      <div className="bg-canvasMuted text-muted flex h-8 w-8 items-center justify-center rounded-full text-xs uppercase">
        {org?.name.substring(0, 2) || '?'}
      </div>
      {!collapsed && (
        <div className="ml-2 flex flex-col justify-start">
          <div className="text-muted leading-1 text-sm">{org?.name}</div>
          <div className="text-subtle text-xs leading-4">
            {user.firstName} {user.lastName}
          </div>
        </div>
      )}
    </div>
  );
};
