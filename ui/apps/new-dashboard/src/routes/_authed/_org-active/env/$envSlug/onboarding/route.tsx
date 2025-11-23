import { Header } from "@inngest/components/Header/NewHeader";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import { pathCreator } from "@/utils/urls";

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/onboarding",
)({
  component: OnboardingLayout,
});

function OnboardingLayout() {
  return (
    <>
      <Header
        breadcrumb={[
          { text: "Getting started", href: pathCreator.onboarding() },
        ]}
      />
      <Outlet />
    </>
  );
}
