import { useEffect, useRef, useState } from "react";

import type { ProfileDisplayType } from "@/queries/server-only/profile";
import Logo from "../Navigation/Logo";
import { Profile } from "../Navigation/Profile";

export default function SideBar({
  collapsed: serverCollapsed,
  profile,
}: {
  collapsed: boolean | undefined;
  profile: ProfileDisplayType;
}) {
  const navRef = useRef<HTMLDivElement>(null);

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
        <div className="mx-4 h-full"></div>
        <Profile collapsed={collapsed} profile={profile} />
      </div>
    </nav>
  );
}
