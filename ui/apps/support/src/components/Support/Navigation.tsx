import { Button } from "@inngest/components/Button/NewButton";
import { InngestLogoSmall } from "@inngest/components/icons/logos/InngestLogoSmall";
import { RiMenuFill, RiAddLine, RiUserLine } from "@remixicon/react";
import { ProfileMenu } from "../Navigation/ProfileMenu";
import { useAuth, useUser, useOrganization } from "@clerk/tanstack-react-start";
import { Image } from "@unpic/react";
import { Link } from "@inngest/components/Link";
import { Link as RouterLink } from "@tanstack/react-router";

export function Navigation() {
  const { isSignedIn } = useAuth();
  const { user } = useUser();
  const { organization } = useOrganization();

  return (
    <>
      {/* Mobile: Horizontal Navigation */}
      <nav className="bg-canvasBase border-subtle sticky top-0 z-50 flex items-center justify-between border-b border-t px-4 py-3 md:hidden">
        {/* Logo */}
        <div className="flex h-8 w-8 shrink-0 items-center justify-center">
          <Link href={import.meta.env.VITE_HOME_PATH}>
            <InngestLogoSmall className="text-basis" />
          </Link>
        </div>

        <div className="flex items-center justify-end gap-6">
          <RouterLink to="/new">
            <Button
              kind="primary"
              appearance="solid"
              size="small"
              label="New ticket"
              className="h-8 px-3 text-sm"
            />
          </RouterLink>

          {/* Menu Icon */}
          <div className="flex items-center justify-center">
            <ProfileMenu
              isAuthenticated={isSignedIn ?? false}
              email={user?.emailAddresses[0].emailAddress}
              organizationName={organization?.name}
              position="below"
            >
              <RiMenuFill className="h-6 w-6" />
            </ProfileMenu>
          </div>
        </div>
      </nav>

      {/* Desktop: Vertical Left-Aligned Sidebar */}
      <nav className="bg-canvasBase border-subtle hidden sticky top-0 left-0 h-screen shrink-0 flex-col items-center justify-between border-r p-4 md:flex">
        {/* Top Section: Logo and New Request Button */}
        <div className="flex flex-col items-center gap-2">
          {/* Logo */}
          <div className="flex mb-2 h-[28px] w-8 shrink-0 items-center justify-center">
            <Link href={import.meta.env.VITE_HOME_PATH}>
              <InngestLogoSmall className="text-basis" />
            </Link>
          </div>

          <RouterLink to="/new">
            <Button
              kind="primary"
              appearance="solid"
              size="small"
              icon={<RiAddLine className="h-[18px] w-[18px]" />}
              className="h-8 w-8 p-0"
              aria-label="New ticket"
              title="New ticket"
            />
          </RouterLink>
        </div>

        {/* Bottom Section: Profile Icon */}
        <div className="flex items-center justify-center">
          <ProfileMenu
            isAuthenticated={isSignedIn ?? false}
            email={user?.emailAddresses[0].emailAddress}
            organizationName={organization?.name}
          >
            <div className="flex h-8 w-8 items-center justify-center overflow-hidden rounded-full">
              {user?.hasImage && user.imageUrl ? (
                <Image
                  src={user.imageUrl}
                  className="h-8 w-8 rounded-full object-cover"
                  width={32}
                  height={32}
                  alt="User avatar"
                />
              ) : organization?.hasImage && organization.imageUrl ? (
                <Image
                  src={organization.imageUrl}
                  className="h-8 w-8 rounded-full object-cover"
                  width={32}
                  height={32}
                  alt="Organization avatar"
                />
              ) : (
                <div className="bg-secondary-moderate flex h-6 w-6 items-center justify-center rounded-full text-alwaysWhite">
                  <RiUserLine className="h-6 w-6" />
                </div>
              )}
            </div>
          </ProfileMenu>
        </div>
      </nav>
    </>
  );
}
