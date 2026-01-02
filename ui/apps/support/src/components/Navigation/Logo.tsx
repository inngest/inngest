import { Button } from "@inngest/components/Button/Button";
import { InngestLogo } from "@inngest/components/icons/logos/InngestLogo";
import { InngestLogoSmall } from "@inngest/components/icons/logos/InngestLogoSmall";
import { RiContractLeftLine, RiContractRightLine } from "@remixicon/react";
import { Link } from "@tanstack/react-router";

type LogoProps = {
  collapsed: boolean;
  setCollapsed: (arg: boolean) => void;
};

const NavToggle = ({
  collapsed,
  setCollapsed,
}: {
  collapsed: boolean;
  setCollapsed: (arg: boolean) => void;
}) => {
  const toggle = () => {
    const toggled = !collapsed;
    setCollapsed(toggled);

    if (typeof window !== "undefined") {
      window.cookieStore.set("navCollapsed", toggled ? "true" : "false");
      //
      // some downstream things, like charts, may need to redraw themselves
      setTimeout(() => window.dispatchEvent(new Event("navToggle")), 200);
    }
  };

  return (
    <Button
      kind="primary"
      appearance="ghost"
      size="small"
      onClick={toggle}
      className={"hidden group-hover:block"}
      icon={
        collapsed ? (
          <RiContractRightLine className="text-muted h-5 w-5" />
        ) : (
          <RiContractLeftLine className="text-muted h-5 w-5" />
        )
      }
    />
  );
};

export default function Logo({ collapsed, setCollapsed }: LogoProps) {
  return (
    <div
      className={`${
        collapsed ? "mx-auto" : "mx-4"
      } mt-4 flex h-[28px] flex-row items-center justify-between`}
    >
      <div
        className={`flex flex-row items-center justify-start ${
          collapsed ? "" : "mr-1"
        } `}
      >
        {collapsed ? (
          <div className="cursor-pointer group-hover:hidden">
            <InngestLogoSmall className="text-basis" />
          </div>
        ) : (
          <Link to={import.meta.env.VITE_HOME_PATH} preload={false}>
            <InngestLogo className="text-basis mr-2" width={96} />
          </Link>
        )}
      </div>
      <NavToggle collapsed={collapsed} setCollapsed={setCollapsed} />
    </div>
  );
}
