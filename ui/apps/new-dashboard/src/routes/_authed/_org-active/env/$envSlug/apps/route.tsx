import { createFileRoute, Outlet } from "@tanstack/react-router";
import { HeaderType } from "@inngest/components/Header/NewHeader";
import { Info } from "@inngest/components/Info/Info";
import { Link } from "@inngest/components/Link/NewLink";

const AppInfo = () => (
  <Info
    text="Apps map directly to your products or services."
    action={
      <Link href="https://www.inngest.com/docs/apps" target="_blank">
        Learn how apps work
      </Link>
    }
  />
);

export const Route = createFileRoute("/_authed/_org-active/env/$envSlug/apps")({
  component: AppLayout,
  beforeLoad: () => {
    return {
      layoutHeader: {
        breadcrumb: [{ text: "Apps" }],
        backNav: true,
        infoIcon: AppInfo(),
      } satisfies HeaderType,
    };
  },
});

function AppLayout() {
  return <Outlet />;
}
