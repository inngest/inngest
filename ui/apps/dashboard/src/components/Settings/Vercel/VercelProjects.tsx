"use client";

import { useState } from "react";
import { Button } from "@inngest/components/Button/NewButton";
import { Card } from "@inngest/components/Card/Card";
import { Info } from "@inngest/components/Info/Info";
import { Link } from "@inngest/components/Link/NewLink";
import { Pill } from "@inngest/components/Pill/NewPill";
import { Select } from "@inngest/components/Select/NewSelect";
import { RiInformationLine, RiRefreshLine } from "@remixicon/react";

import {
  VercelDeploymentProtection,
  type VercelIntegration,
} from "@/gql/graphql";

export function VercelProjects({
  integration,
}: {
  integration: VercelIntegration;
}) {
  const { projects } = integration;
  const [filter, setFilter] = useState("all");

  return (
    <div className="mt-8 flex flex-col">
      <div className="flex flex-row items-center justify-between">
        <div className="text-muted">
          Projects (<span className="mx-[2px]">{projects.length}</span>)
        </div>
        <div className="text-btnPrimary flex cursor-pointer flex-row items-center justify-between text-xs">
          <Button
            appearance="ghost"
            icon={<RiRefreshLine className="h-4 w-4" />}
            iconSide="left"
            label="Refresh list"
          />

          <Select
            value={{ id: "all", name: "All" }}
            onChange={(o) => setFilter(o.name)}
            label="Show"
            className="text-muted bg-canvasBase ml-4 h-6 rounded-sm text-xs leading-tight"
          >
            <Select.Button className="rounded-0 h-4">
              <span className="text-basis pr-2 text-xs leading-tight first-letter:capitalize">
                {filter}
              </span>
            </Select.Button>
            <Select.Options>
              {["all", "disabled", "enabled"].map((o, i) => {
                return (
                  <Select.Option
                    key={`option-${i}`}
                    option={{ id: o, name: o }}
                  >
                    <span className="inline-flex w-full items-center justify-between gap-2">
                      <label className="text-sm lowercase first-letter:capitalize">
                        {o}
                      </label>
                    </span>
                  </Select.Option>
                );
              })}
            </Select.Options>
          </Select>
        </div>
      </div>
      {projects
        .filter((p) =>
          filter === "all"
            ? true
            : filter === "enabled"
            ? p.isEnabled
            : !p.isEnabled,
        )
        .map((p, i) => (
          <Card
            key={`vercel-projects-${i}`}
            className="mt-4"
            accentPosition="left"
            accentColor={p.isEnabled ? "bg-primary-intense" : "bg-surfaceMuted"}
          >
            <Card.Content className="h-36 p-6">
              <div className="flex flex-row items-center justify-between">
                <div className="flex flex-col">
                  <div>
                    <Pill
                      kind={p.isEnabled ? "primary" : "default"}
                      className="h-6"
                    >
                      {p.isEnabled ? "enabled" : "disabled"}
                    </Pill>
                  </div>
                  <div className="mt-4 flex flex-row items-center justify-start gap-1">
                    <div className="text-basis text-xl font-medium">
                      {p.name}
                    </div>
                    {p.deploymentProtection !==
                      VercelDeploymentProtection.Disabled && (
                      <Info
                        text={
                          <>
                            <div className="text-sm font-medium">
                              Deployment protection is enabled
                            </div>
                            <div className="mt-2 text-sm font-normal">
                              Inngest may not be able to communicate with your
                              application by default.
                            </div>
                          </>
                        }
                        action={
                          <Link
                            target="_blank"
                            href="https://www.inngest.com/docs/deploy/vercel#bypassing-deployment-protection"
                          >
                            Learn more
                          </Link>
                        }
                        iconElement={
                          <RiInformationLine className="text-subtle h-[18px] w-[18px]" />
                        }
                      />
                    )}
                  </div>
                  <div className="text-muted mt-2 text-base font-normal leading-snug">
                    {p.servePath}
                  </div>
                </div>
                <div>
                  <Button
                    appearance="outlined"
                    label="Configure"
                    href={`/settings/integrations/vercel/configure/${encodeURIComponent(
                      p.projectID,
                    )}`}
                  />
                </div>
              </div>
            </Card.Content>
          </Card>
        ))}
    </div>
  );
}
