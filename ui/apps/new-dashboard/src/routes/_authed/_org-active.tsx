import { getEnv } from "@/components/Environments/data";
import SideBar from "@/components/Layout/SideBar";
import URQLProviderWrapper from "@/components/URQL/URQLProvider";
import { navCollapsed } from "@/data/nav";
import { createFileRoute, Outlet, useParams } from "@tanstack/react-router";

export const Route = createFileRoute("/_authed/_org-active")({
  component: OrgActive,

  head: () => ({
    //
    // TANSTACK TODO: initialize maze here
    scripts: [
      {
        src: "",
        type: "text/javascript",
      },
    ],
  }),
  loader: async ({ params }: { params: { env?: string } }) => {
    const env = params.env ? await getEnv(params.env) : undefined;
    return {
      env,
      navCollapsed: await navCollapsed(),
    };
  },
});

function OrgActive() {
  const { navCollapsed, env: env } = Route.useLoaderData();
  console.log("got env", env);

  return (
    <URQLProviderWrapper>
      <div
        id="layout-scroll-container"
        className="fixed z-50 flex h-screen w-full flex-row justify-start overflow-y-scroll overscroll-y-none"
      >
        {/* <SideBar activeEnv={env} collapsed={navCollapsed} profile={undefined} /> */}

        <div className="no-scrollbar flex w-full flex-col overflow-x-scroll">
          {/* TANSTACK TODO: add incident banner, billing banner, and execution overage banner here */}

          <div className="flex-col">
            <div className="no-scrollbar overflow-y-scroll px-6">
              <Outlet />
            </div>
          </div>
        </div>
      </div>
    </URQLProviderWrapper>
  );
}
