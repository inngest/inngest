import { Button } from '@inngest/components/Button';
import { RiContractLeftLine, RiContractRightLine } from '@remixicon/react';

type LogoProps = {
  collapsed: boolean;
  setCollapsed: (arg: boolean) => void;
};

// Sidebar top — used to host the Inngest mark, now just the hover-revealed
// collapse toggle. Kept as a component to preserve the top-of-sidebar slot.
export default function Logo({ collapsed, setCollapsed }: LogoProps) {
  const toggle = async () => {
    const toggled = !collapsed;
    setCollapsed(toggled);

    if (typeof window !== 'undefined') {
      window.cookieStore.set('navCollapsed', toggled ? 'true' : 'false');
      // some downstream things, like charts, may need to redraw themselves
      setTimeout(() => window.dispatchEvent(new Event('navToggle')), 200);
    }
  };

  return (
    <div
      className={`${
        collapsed ? 'mx-auto' : 'mx-4'
      } mt-3 flex h-6 flex-row items-center justify-end`}
    >
      <Button
        kind="primary"
        appearance="ghost"
        size="small"
        onClick={toggle}
        className="hidden group-hover:block"
        icon={
          collapsed ? (
            <RiContractRightLine className="text-muted h-5 w-5" />
          ) : (
            <RiContractLeftLine className="text-muted h-5 w-5" />
          )
        }
      />
    </div>
  );
}
