import type { Route } from "next";
import { redirect } from "next/navigation";
import { Button } from "@inngest/components/Button/index";
import { Link } from "@inngest/components/Link/Link";
import { IconVercel } from "@inngest/components/icons/platforms/Vercel";
import { RiExternalLinkLine } from "@remixicon/react";

import { pathCreator } from "@/utils/urls";
import { getVercelIntegration } from "../data";
import VercelProjects from "./projects";

export const dynamic = "force-dynamic";
export const fetchCache = "force-no-store";

export default async function VercelIntegrationPage() {
  const integration = await getVercelIntegration();
  if (!integration) {
    console.error("Failed to load Vercel integration");
    return redirect(pathCreator.vercelSetup());
  }
  if (integration instanceof Error) {
    throw integration;
  }

  return (
    <div className="mx-auto mt-6 flex w-[800px] flex-col p-8">
      <div className="mt-6 flex flex-row items-center justify-between">
        <div className="flex flex-row items-center justify-start">
          <div className="bg-contrast mb-7 mr-4 flex h-[52px] w-[52px] items-center justify-center rounded">
            <IconVercel className="text-onContrast h-6 w-6" />
          </div>
          <div className="flex flex-col">
            <div className="text-basis mb-1 text-xl font-medium leading-7">
              Vercel
            </div>
            <div className="text-muted mb-7 text-base">
              You can manage all your projects on this page.{" "}
              <Link
                size="medium"
                href={"https://www.inngest.com/docs/deploy/vercel" as Route}
                target="_blank"
                className="inline"
              >
                Learn more
              </Link>
            </div>
          </div>
        </div>

        <div className="place-self-start">
          <Button
            appearance="outlined"
            kind="secondary"
            href={"https://vercel.com/integrations/inngest" as Route}
            icon={<RiExternalLinkLine className="mr-1" />}
            iconSide="left"
            label="Go to Vercel"
          />
        </div>
      </div>
      <VercelProjects integration={integration} />
    </div>
  );
}
