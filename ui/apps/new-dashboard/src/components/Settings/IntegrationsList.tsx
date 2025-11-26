import type { ReactNode } from "react";
import { Button } from "@inngest/components/Button/NewButton";
import { Card } from "@inngest/components/Card/Card";
import { IconDatadog } from "@inngest/components/icons/platforms/Datadog";
import { IconNeon } from "@inngest/components/icons/platforms/Neon";
import { IconNetlify } from "@inngest/components/icons/platforms/Netlify";
import { IconPrometheus } from "@inngest/components/icons/platforms/Prometheus";
import { IconSupabase } from "@inngest/components/icons/platforms/Supabase";
import { IconVercel } from "@inngest/components/icons/platforms/Vercel";
import { RiExternalLinkLine } from "@remixicon/react";

import { useBooleanFlag } from "@/components/FeatureFlags/hooks";

type Integration = {
  title: string;
  slug: string;
  Icon: ReactNode;
  actionButton: (args: {
    enabled: boolean;
    hasError?: boolean;
    isMarketplace?: boolean;
  }) => ReactNode;
  description: string;
  upvoteId?: string;
};

const INTEGRATIONS: Integration[] = [
  {
    title: "Vercel",
    slug: "vercel",
    Icon: <IconVercel className="text-onContrast h-6 w-6" />,
    actionButton: ({ enabled, hasError }) => {
      let label: string;
      let target = undefined;
      let url: string;
      if (enabled) {
        label = "Manage";
        url = "/settings/integrations/vercel";
      } else {
        //
        // Integration not installed.
        label = "Connect";
        url = "/settings/integrations/vercel/connect";
      }

      return (
        <Button
          disabled={hasError}
          kind="primary"
          appearance={enabled ? "outlined" : "solid"}
          size="medium"
          //@ts-expect-error TANSTACK TODO: remove after this route lands
          to={url}
          label={label}
          target={target}
        />
      );
    },
    description:
      "Host your Inngest functions on Vercel and automatically sync them every time you deploy code.",
  },
  {
    title: "Neon",
    slug: "neon",
    Icon: <IconNeon className="text-onContrast h-6 w-6" />,
    actionButton: ({ enabled, hasError }) => (
      <Button
        disabled={hasError}
        kind="primary"
        appearance={enabled ? "outlined" : "solid"}
        size="medium"
        to={
          enabled
            ? "/settings/integrations/neon"
            : "/settings/integrations/neon/connect"
        }
        label={enabled ? "Manage" : "Connect"}
      />
    ),
    description:
      "Connect to send events directly from changes in your Neon database.",
  },
  {
    title: "Supabase",
    slug: "supabase",
    Icon: <IconSupabase className="text-onContrast h-6 w-6" />,
    actionButton: ({ enabled, hasError }) => (
      <Button
        disabled={hasError}
        kind="primary"
        appearance={enabled ? "outlined" : "solid"}
        size="medium"
        to={
          enabled
            ? "/settings/integrations/supabase"
            : "/settings/integrations/supabase/connect"
        }
        label={enabled ? "Manage" : "Connect"}
      />
    ),
    description:
      "Connect to send events directly from changes in your Subapase database.",
  },
  {
    title: "Netlify",
    slug: "netlify",
    Icon: <IconNetlify className="text-onContrast h-6 w-6" />,
    actionButton: () => (
      <Button
        icon={<RiExternalLinkLine />}
        iconSide="left"
        kind="secondary"
        appearance="outlined"
        size="medium"
        label="View docs"
        href="https://www.inngest.com/docs/deploy/netlify"
      />
    ),
    description:
      "Check out our docs to see how you can use Inngest with your applications deployed to Netlify.",
  },
  {
    title: "Prometheus",
    slug: "prometheus",
    Icon: <IconPrometheus className="text-onContrast h-6 w-6" />,
    actionButton: () => (
      <Button
        iconSide="left"
        appearance="outlined"
        size="medium"
        label="Configure"
        to="/settings/integrations/prometheus"
      />
    ),
    description:
      "Configure your Prometheus instance to scrape metrics directly from Inngest.",
  },
  {
    title: "Datadog",
    slug: "datadog",
    Icon: <IconDatadog className="text-onContrast h-6 w-6" />,
    actionButton: () => (
      <Button
        iconSide="left"
        appearance="solid"
        size="medium"
        label="Connect"
        to="/settings/integrations/datadog"
      />
    ),
    description: "Send metrics from Inngest directly to Datadog",
  },
];

type Props = {
  integrations: {
    error?: string;
    slug: string;
    enabled: boolean;
    isMarketplace?: boolean;
    projects: {}[];
  }[];
};

export const IntegrationsList = ({ integrations }: Props) => {
  const { value: supabaseIntegration } = useBooleanFlag("supabase-integration");

  const getIntegrationData = (slug: string) =>
    integrations.find((integration) => integration.slug === slug);

  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="mb-7 w-full text-2xl font-medium">All integrations</div>
      <div className="grid w-[800px] grid-cols-2 gap-6">
        {INTEGRATIONS.map((i: Integration, n) => {
          if (i.title === "Supabase" && !supabaseIntegration) return;

          const integrationData = getIntegrationData(i.slug);
          const isEnabled = Boolean(integrationData?.enabled);
          const isMarketplace = Boolean(integrationData?.isMarketplace);

          return (
            <Card key={`integration-card-${n}`}>
              <div className="flex w-[388px] flex-col p-6">
                <div className="align-center flex flex-row items-center justify-between">
                  <div className="bg-contrast flex h-12 w-12 items-center justify-center rounded">
                    {i.Icon}
                  </div>

                  {i.actionButton({
                    enabled: isEnabled,
                    hasError: Boolean(integrationData?.error),
                    isMarketplace,
                  })}
                </div>
                <div className="text-basis mt-[18px] text-lg font-medium">
                  {i.title}
                </div>
                <div className="text-muted mt-2 text-sm leading-tight">
                  {i.description}
                </div>
                {integrationData?.error && (
                  <div className="text-error mt-2 text-sm leading-tight">
                    {integrationData.error}
                  </div>
                )}
              </div>
            </Card>
          );
        })}
        <Card className="col-span-2">
          <div className="bg-canvasSubtle flex w-[800px] flex-col p-6">
            <div className="text-basis text-lg font-medium">
              Can&apos;t find what you need?
            </div>
            <div className="text-basis mt-3 text-sm leading-tight">
              Write to our team about the integration you are looking for and we
              will get back to you.
            </div>
            <div>
              <Button
                icon={<RiExternalLinkLine />}
                iconSide="left"
                kind="secondary"
                appearance="outlined"
                size="medium"
                label="Request integration"
                className="border-muted bg-subtle mt-5"
                href="https://roadmap.inngest.com/roadmap"
                target="_blank"
              />
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
};
