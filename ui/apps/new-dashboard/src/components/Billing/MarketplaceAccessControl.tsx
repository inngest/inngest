import { useNavigate, useLocation } from "@tanstack/react-router";
import { useEffect } from "react";

import { pathCreator } from "@/utils/urls";

//
// Whitelist of paths that marketplace users can access
const marketplaceAllowedPaths = ["/usage"] as const;

type Props = {
  isMarketplace: boolean;
};

export const MarketplaceAccessControl = ({
  children,
  isMarketplace,
}: React.PropsWithChildren<Props>) => {
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    if (isMarketplace) {
      const isAllowed = marketplaceAllowedPaths.some((allowedPath) =>
        location.pathname.endsWith(allowedPath),
      );

      if (!isAllowed) {
        //
        // Redirect to usage page if trying to access non-whitelisted page
        navigate({ to: pathCreator.billing({ tab: "usage" }) });
      }
    }
  }, [isMarketplace, location.pathname, navigate]);

  return <>{children}</>;
};
