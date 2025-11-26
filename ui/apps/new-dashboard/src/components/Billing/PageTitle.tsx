import { useLocation } from "@tanstack/react-router";
import { Link } from "@inngest/components/Link/NewLink";

import { WEBSITE_PRICING_URL, pathCreator } from "@/utils/urls";

export default function PageTitle() {
  const location = useLocation();
  const pathname = location.pathname;

  const routeTitles: { [key: string]: string } = {
    [pathCreator.billing()]: "Overview",
    [pathCreator.billing({ tab: "usage" })]: "Usage",
    [pathCreator.billing({ tab: "payments" })]: "Payments",
    [pathCreator.billing({ tab: "plans" })]: "Plans",
  };
  const pageTitle = routeTitles[pathname] || "";
  const cta =
    pathname === pathCreator.billing({ tab: "plans" }) ? (
      <Link
        target="_blank"
        size="small"
        href={WEBSITE_PRICING_URL + "?ref=app-billing-page-plans"}
      >
        View pricing page
      </Link>
    ) : null;

  return (
    <div className="text-basis flex items-center justify-between">
      <h2 className="my-9 text-2xl">{pageTitle}</h2>
      {cta}
    </div>
  );
}
