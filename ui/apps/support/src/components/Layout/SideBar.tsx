import { useRef } from "react";
import { Link, useLocation } from "@tanstack/react-router";
import { RiTicketLine } from "@remixicon/react";

import Logo from "../Navigation/Logo";
import { Profile } from "../Navigation/Profile";
import { ProfileDisplayType } from "@/data/profile";

export default function SideBar({
  profile,
}: {
  collapsed?: boolean | undefined;
  profile?: ProfileDisplayType;
}) {
  const navRef = useRef<HTMLDivElement>(null);
  const location = useLocation();
  const pathname = location.pathname;

  // Always collapsed in the new design
  const collapsed = true;

  const navItems = [
    {
      icon: RiTicketLine,
      label: "Tickets",
      path: "/",
      exact: true,
    },
  ];

  return (
    <nav
      className="bg-canvasBase border-muted sticky top-0 z-[51] flex h-screen w-[59px] shrink-0 flex-col justify-start overflow-visible border-r"
      ref={navRef}
    >
      <Logo collapsed={collapsed} setCollapsed={() => {}} />
      <div className="flex grow flex-col justify-between">
        <div className="mx-2 mt-2 space-y-2">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = item.exact
              ? pathname === item.path
              : pathname.startsWith(item.path);
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`flex h-8 w-8 items-center justify-center rounded-md text-sm font-medium transition-colors ${
                  isActive
                    ? "bg-primary-moderate text-alwaysWhite"
                    : "text-subtle hover:bg-canvasSubtle hover:text-basis"
                }`}
                title={item.label}
              >
                <Icon className="h-[18px] w-[18px] shrink-0" />
              </Link>
            );
          })}
        </div>
        {profile && <Profile collapsed={collapsed} profile={profile} />}
      </div>
    </nav>
  );
}
