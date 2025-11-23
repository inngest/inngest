import { Header } from "@inngest/components/Header/NewHeader";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/debugger/$functionSlug/",
)({
  component: DebuggerFunctionPage,
});

function DebuggerFunctionPage() {
  const { functionSlug } = Route.useParams();

  return (
    <>
      <Header
        breadcrumb={[
          { text: "Runs" },
          { text: functionSlug },
          { text: "Debug" },
        ]}
        action={<div className="flex flex-row items-center gap-x-1"></div>}
      />
      <div>coming soon...</div>
    </>
  );
}
