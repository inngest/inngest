import { Header } from "@inngest/components/Header/NewHeader";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/debugger/",
)({
  component: DebuggerPage,
});

function DebuggerPage() {
  return (
    <>
      <Header breadcrumb={[{ text: "Debug" }]} />
      <div>coming soon...</div>
    </>
  );
}
