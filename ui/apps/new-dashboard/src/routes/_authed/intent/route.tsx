import { InngestLogo } from "@inngest/components/icons/logos/InngestLogo";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import { getProfileDisplay } from "@/queries/server/profile";

export const Route = createFileRoute("/_authed/intent")({
  component: IntenComponent,
  loader: async () => {
    const profile = await getProfileDisplay();
    return { profile };
  },
});

function IntenComponent() {
  const { profile } = Route.useLoaderData();

  return (
    <div className="h-full overflow-y-scroll">
      <div className="mx-auto flex h-full max-w-screen-xl flex-col px-6">
        <header className="flex items-center justify-between py-6">
          <div>
            <InngestLogo />
            <h1 className="hidden">Inngest</h1>
          </div>
          <div className="flex items-center gap-2">
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
            <p>{profile.orgName}</p>
          </div>
        </header>
        <div className="flex grow items-center">
          <Outlet />
        </div>
      </div>
    </div>
  );
}
