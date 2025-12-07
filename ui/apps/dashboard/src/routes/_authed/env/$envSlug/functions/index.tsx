import { useMemo } from "react";

import { Button } from "@inngest/components/Button/NewButton";
import { FunctionsTable } from "@inngest/components/Functions/NewFunctionsTable";
import { Header } from "@inngest/components/Header/NewHeader";
import { RiExternalLinkLine, RiRefreshLine } from "@remixicon/react";

import { FunctionInfo } from "@/components/Functions/FunctionInfo";
import {
  useFunctionVolume,
  useFunctions,
} from "@/components/Functions/useFunctions";
import { pathCreator } from "@/utils/urls";
import {
  useNavigate,
  createFileRoute,
  ClientOnly,
} from "@tanstack/react-router";

export const Route = createFileRoute("/_authed/env/$envSlug/functions/")({
  component: FunctionPage,
});

function FunctionPage() {
  const { envSlug } = Route.useParams();
  const navigate = useNavigate();
  const internalPathCreator = useMemo(() => {
    return {
      // The shared component library is environment-agnostic, so it needs a way to
      // generate URLs without knowing about environments
      function: (params: { functionSlug: string }) =>
        pathCreator.function({
          envSlug: envSlug,
          functionSlug: params.functionSlug,
        }),
      eventType: (params: { eventName: string }) =>
        pathCreator.eventType({
          envSlug: envSlug,
          eventName: params.eventName,
        }),
      app: (params: { externalAppID: string }) =>
        pathCreator.app({
          envSlug: envSlug,
          externalAppID: params.externalAppID,
        }),
    };
  }, [envSlug]);
  const getFunctions = useFunctions();
  const getFunctionVolume = useFunctionVolume();

  return (
    <>
      <Header
        breadcrumb={[{ text: "Functions" }]}
        infoIcon={<FunctionInfo />}
      />
      <ClientOnly>
        <FunctionsTable
          pathCreator={internalPathCreator}
          getFunctions={getFunctions}
          getFunctionVolume={getFunctionVolume}
          emptyActions={
            <>
              <Button
                appearance="outlined"
                label="Refresh"
                onClick={() => navigate({ to: "." })}
                icon={<RiRefreshLine />}
                iconSide="left"
              />
              <Button
                label="Go to docs"
                href="https://www.inngest.com/docs/learn/inngest-functions"
                target="_blank"
                icon={<RiExternalLinkLine />}
                iconSide="left"
              />
            </>
          }
        />
      </ClientOnly>
    </>
  );
}
