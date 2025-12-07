import { UserProfile } from "@clerk/tanstack-react-start";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authed/settings/user/")({
  component: UserSettingsPage,
});

function UserSettingsPage() {
  return (
    <div className="flex flex-col justify-start">
      <UserProfile
        routing="path"
        path="/settings/user"
        appearance={{
          layout: {
            logoPlacement: "none",
          },
          elements: {
            navbar: "hidden",
            scrollBox: "bg-canvasBase shadow-none",
            pageScrollBox: "pt-6 px-2",
          },
        }}
      >
        <UserProfile.Page label="security" />
      </UserProfile>
      <UserProfile
        routing="path"
        path="/settings/security"
        appearance={{
          layout: {
            logoPlacement: "none",
          },
          elements: {
            navbar: "hidden",
            scrollBox: "bg-canvasBase shadow-none",
            pageScrollBox: "pt-0 px-2",
          },
        }}
      >
        <UserProfile.Page label="account" />
      </UserProfile>
    </div>
  );
}
