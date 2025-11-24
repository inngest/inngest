import { Header } from "@inngest/components/Header/NewHeader";
import { createFileRoute } from "@tanstack/react-router";

import { CreateEnvironment } from "@/components/CreateEnvironment/CreateEnvironment";

export const Route = createFileRoute("/_authed/create-environment/")({
  component: CreateEnvironmentPage,
});

function CreateEnvironmentPage() {
  return (
    <>
      <Header
        breadcrumb={[
          { text: "Environments", href: "/env" },
          { text: "Create" },
        ]}
      />
      <div className="no-scrollbar overflow-y-scroll p-6">
        <CreateEnvironment />
      </div>
    </>
  );
}
