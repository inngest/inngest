import { useLocation } from "@tanstack/react-router";
import { ProfileMenu } from "./ProfileMenu";
import { ProfileDisplayType } from "@/queries/server-only/profile";

export const Profile = ({
  collapsed,
  profile,
}: {
  collapsed: boolean;
  profile: ProfileDisplayType;
}) => {
  const location = useLocation();
  const pathname = location.pathname;
  const active =
    pathname.startsWith("/settings/organization") ||
    pathname.startsWith("/billing") ||
    pathname.startsWith("/settings/user");

  return (
    <ProfileMenu isMarketplace={profile.isMarketplace}>
      <div
        className={`border-subtle mt-2 flex h-16 w-full flex-row items-center justify-start border-t px-2.5 `}
      >
        <div
          className={`flex w-full flex-row items-center rounded p-1 ${
            collapsed ? "justify-center" : "justify-start"
          } ${
            active
              ? "bg-secondary-4xSubtle text-info hover:bg-secondary-3xSubtle"
              : "hover:bg-canvasSubtle text-subtle"
          }`}
        >
          <div className="bg-canvasMuted text-subtle flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs uppercase">
            {profile.orgProfilePic ? (
              <img
                src={profile.orgProfilePic}
                className="h-8 w-8 rounded-full object-cover"
                width={32}
                height={32}
                alt="org-profile-pic"
              />
            ) : (
              profile.orgName?.substring(0, 2) || "?"
            )}
          </div>

          {!collapsed && (
            <div className="ml-2 flex flex-col items-start justify-start overflow-hidden">
              <div
                className="text-subtle leading-1 max-w-full truncate text-sm"
                title={profile.orgName}
              >
                {profile.orgName}
              </div>
              <div
                className="text-muted max-w-full truncate text-xs leading-4"
                title={profile.displayName}
              >
                {profile.displayName}
              </div>
            </div>
          )}
        </div>
      </div>
    </ProfileMenu>
  );
};
