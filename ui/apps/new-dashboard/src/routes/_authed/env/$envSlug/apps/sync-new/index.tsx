import { Button } from "@inngest/components/Button/NewButton";
import { CodeLine } from "@inngest/components/CodeLine";
import { Link } from "@inngest/components/Link/NewLink";
import TabCards from "@inngest/components/TabCards/TabCards";
import { IconVercel } from "@inngest/components/icons/platforms/Vercel";
import { RiCodeLine, RiInputCursorMove } from "@remixicon/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { ManualSync } from "@/components/Apps/SyncNew/ManualSync";
import { pathCreator } from "@/utils/urls";

export const Route = createFileRoute("/_authed/env/$envSlug/apps/sync-new/")({
  component: SyncNewPage,
});

function SyncNewPage() {
  const { envSlug } = Route.useParams();
  const navigate = useNavigate();
  const appsURL = pathCreator.apps({ envSlug });

  return (
    <div className="bg-canvasBase text-basis h-full overflow-y-auto">
      <section className="mx-auto mt-16 max-w-screen-md text-sm">
        <h2 className="text-basis mb-4 text-xl">Sync your app to Inngest</h2>
        <p className="mb-6 text-sm">
          Since your code is hosted on another platform, you need to register
          where your functions are hosted with Inngest.{" "}
          <Link
            className="inline-block"
            size="small"
            href="https://www.inngest.com/docs/apps/cloud?ref=app-onboarding-sync-app"
            target="_blank"
          >
            Learn more about syncs
          </Link>
        </p>

        <h4 className="mb-4 text-sm font-medium">Choose syncing method:</h4>
        <TabCards defaultValue="tab1">
          <TabCards.ButtonList>
            <TabCards.Button className="w-36" value="tab1">
              <div className="flex items-center gap-1.5">
                <RiInputCursorMove className="h-5 w-5" /> Sync manually
              </div>
            </TabCards.Button>
            <TabCards.Button className="w-36" value="tab2">
              <div className="flex items-center gap-1.5">
                <IconVercel className="h-4 w-4" /> Sync with Vercel
              </div>
            </TabCards.Button>
            <TabCards.Button className="w-36" value="tab3">
              <div className="flex items-center gap-1.5">
                <RiCodeLine className="h-5 w-5" /> Curl command
              </div>
            </TabCards.Button>
          </TabCards.ButtonList>

          <TabCards.Content value="tab1">
            <ManualSync appsURL={appsURL} />
          </TabCards.Content>
          <TabCards.Content value="tab2">
            <p>
              Inngest enables you to host your apps on Vercel using their
              serverless functions platform. By using Inngest&apos;s official
              Vercel integration, your apps will be synced automatically.
            </p>
            <div className="flex items-center justify-between pt-6">
              <Link
                href="https://www.inngest.com/docs/apps/cloud"
                target="_blank"
                size="small"
              >
                View Docs
              </Link>
              <div className="flex items-center gap-3">
                <Button
                  label="Cancel"
                  onClick={() => {
                    navigate({ to: appsURL });
                  }}
                  appearance="outlined"
                  kind="secondary"
                />
                <Button
                  label="Go to Vercel configuration"
                  href="/settings/integrations/vercel"
                  kind="primary"
                />
              </div>
            </div>
          </TabCards.Content>
          <TabCards.Content value="tab3">
            <div className="">
              <p>
                You can sync from your machine or automate this within a CI/CD
                pipeline.
                <span className="font-semibold">
                  {" "}
                  Send a PUT request to your own application&apos;s serve
                  endpoint using the following command:
                </span>
              </p>
              <CodeLine
                code="curl -X PUT https://<your-app>.com/api/inngest"
                className="mt-6"
              />
            </div>
            <div className="flex items-center justify-between pt-6">
              <Link
                href="https://www.inngest.com/docs/apps/cloud"
                target="_blank"
                size="small"
              >
                View Docs
              </Link>
              <div className="flex items-center gap-3">
                <Button
                  label="Done"
                  onClick={() => {
                    navigate({ to: appsURL });
                  }}
                  kind="primary"
                />
              </div>
            </div>
          </TabCards.Content>
        </TabCards>
      </section>
    </div>
  );
}
