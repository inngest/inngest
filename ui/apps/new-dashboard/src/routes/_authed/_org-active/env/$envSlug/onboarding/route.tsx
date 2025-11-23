import { Header } from "@inngest/components/Header/NewHeader";
import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

import { pathCreator } from "@/utils/urls";

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/onboarding",
)({
  component: OnboardingLayout,
  loader: ({ params }) => {
    if (!("step" in params)) {
      redirect({
        to: "/env/$envSlug/onboarding/$step",
        params: { envSlug: params.envSlug, step: "create-app" },
      });
    }
  },
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
