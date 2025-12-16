import { useEffect, useRef, useState } from "react";
import { Link, useLocation } from "@tanstack/react-router";
import { RiTicketLine, RiHomeLine } from "@remixicon/react";

import Logo from "../Navigation/Logo";
import { Profile } from "../Navigation/Profile";
import { ProfileDisplayType } from "@/data/profile";

export default function SideBar({
  collapsed: serverCollapsed,
  profile,
}: {
  collapsed: boolean | undefined;
  profile?: ProfileDisplayType;
}) {
  const navRef = useRef<HTMLDivElement>(null);
  const location = useLocation();
  const pathname = location.pathname;

  const [collapsed, setCollapsed] = useState<boolean>(serverCollapsed ?? false);

  const autoCollapse = () =>
    typeof window !== "undefined" &&
    window.matchMedia("(max-width: 800px)").matches &&
    setCollapsed(true);

  useEffect(() => {
    //
    // if the user has not set a pref and they are on mobile, collapse by default
    serverCollapsed === undefined && autoCollapse();

    if (navRef.current !== null) {
      window.addEventListener("resize", autoCollapse);

      return () => {
        window.removeEventListener("resize", autoCollapse);
      };
    }
  }, [serverCollapsed]);

  const navItems = [
    {
      icon: RiHomeLine,
      label: "Tickets",
      path: "/",
      exact: true,
    },
    {
      icon: RiTicketLine,
      label: "Support",
      path: "/support",
      exact: false,
    },
  ];

  return (
    <nav
      className={`bg-canvasBase border-subtle group
         top-0 flex h-screen flex-col justify-start ${
           collapsed ? "w-[64px]" : "w-[224px]"
         }  sticky z-[51] shrink-0 overflow-visible border-r`}
      ref={navRef}
    >
      <Logo collapsed={collapsed} setCollapsed={setCollapsed} />
      <div className="flex grow flex-col justify-between">
        <div className="mx-2 mt-4 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = item.exact
              ? pathname === item.path
              : pathname.startsWith(item.path);
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  collapsed ? "justify-center" : ""
                } ${
                  isActive
                    ? "bg-secondary-4xSubtle text-info"
                    : "text-subtle hover:bg-canvasSubtle hover:text-basis"
                }`}
                title={collapsed ? item.label : undefined}
              >
                <Icon className="h-5 w-5 shrink-0" />
                {!collapsed && <span>{item.label}</span>}
              </Link>
            );
          })}
        </div>
        {profile && <Profile collapsed={collapsed} profile={profile} />}
      </div>
    </nav>
  );
}
