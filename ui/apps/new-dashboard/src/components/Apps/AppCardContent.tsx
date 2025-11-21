import { Link } from "@inngest/components/Link/Link";
import {
  HorizontalPillList,
  Pill,
  PillContent,
} from "@inngest/components/Pill";
import { methodTypes, type AppKind } from "@inngest/components/types/app";
import { RiExternalLinkLine } from "@remixicon/react";

import { syncKind, syncStatusText } from "@/components/SyncStatusPill";
import { pathCreator } from "@/utils/urls";
import type { FlattenedApp } from "./useApps";

const getAppCardContent = ({
  app,
  envSlug,
}: {
  app: FlattenedApp;
  envSlug: string;
}) => {
  const statusKey = app.status ?? "default";
  const appKind: AppKind = app.isArchived
    ? "default"
    : // API apps don't currently have sync status, consider them green for now
    app.method === methodTypes.Api
    ? "primary"
    : syncKind[statusKey] ?? "default";

  const status = app.isArchived
    ? "Archived"
    : syncStatusText[statusKey] ?? null;

  const footerHeaderTitle = app.error ? (
    `Error: ${app.error}`
  ) : app.functionCount === 0 ? (
    "There are currently no functions registered at this URL."
  ) : (
    <>
      {app.functionCount} {app.functionCount === 1 ? "function" : "functions"}{" "}
      found
    </>
  );

  const footerHeaderSecondaryCTA =
    !app.error && app.functionCount > 0 ? (
      <Link size="small" href={pathCreator.functions({ envSlug: envSlug })}>
        View functions
      </Link>
    ) : null;

  const footerContent =
    app.functionCount === 0 ? (
      <>
        <p className="text-subtle pb-4">
          Ensure you have created a function and are exporting it correctly from
          your serve() command.
        </p>
        <Link
          size="small"
          target="_blank"
          href="https://www.inngest.com/docs/learn/serving-inngest-functions?ref=cloud-app"
          iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
        >
          How to serve functions
        </Link>
      </>
    ) : (
      <table className="w-full">
        <tbody className="divide-subtle divide-y">
          {[...app.functions]
            .sort((a, b) => a.name.localeCompare(b.name))
            .map((func) => {
              return (
                <tr
                  key={func.id}
                  className="bg-canvaseBase hover:bg-canvasSubtle/50"
                >
                  <td className="py-2">
                    <Link
                      href={pathCreator.function({
                        envSlug,
                        functionSlug: func.slug,
                      })}
                    >
                      {func.name}
                    </Link>
                  </td>
                  <td className="py-2">
                    <HorizontalPillList
                      alwaysVisibleCount={2}
                      pills={func.triggers.map((trigger) => {
                        return (
                          <Pill
                            appearance="outlined"
                            href={
                              trigger.type === "EVENT"
                                ? pathCreator.eventType({
                                    envSlug,
                                    eventName: trigger.value,
                                  })
                                : undefined
                            }
                            key={trigger.type + trigger.value}
                          >
                            <PillContent type={trigger.type}>
                              {trigger.value}
                            </PillContent>
                          </Pill>
                        );
                      })}
                    />
                  </td>
                </tr>
              );
            })}
        </tbody>
      </table>
    );

  return {
    appKind,
    status,
    footerHeaderTitle,
    footerHeaderSecondaryCTA,
    footerContent,
  };
};

export default getAppCardContent;
