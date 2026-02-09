import { RiSlackLine } from "@remixicon/react";
import { getDashboardBillingUrl } from "@/utils/urls";

type SlackChannelUpsellProps = {
  hasPremiumSupport: boolean;
  isEnterprise: boolean;
};

export function SlackChannelUpsell({
  hasPremiumSupport,
  isEnterprise,
}: SlackChannelUpsellProps) {
  // Don't show if user has premium support or enterprise
  if (hasPremiumSupport || isEnterprise) {
    return null;
  }

  return (
    <div className="mt-8 p-4 border rounded flex items-center gap-3 max-w-xl bg-canvasSubtle">
      <RiSlackLine className="text-muted h-5 w-5 shrink-0" />
      <p className="text-sm text-muted">
        Dedicated Slack channels are available with the{" "}
        <a
          href={getDashboardBillingUrl()}
          className="text-link hover:underline"
        >
          premium support add-on
        </a>{" "}
        and enterprise plans.
      </p>
    </div>
  );
}
