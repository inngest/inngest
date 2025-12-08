"use client";

import { Button } from "@inngest/components/Button";
import { RiExternalLinkLine } from "@remixicon/react";

import { SHOW_EXAMPLES_LINK } from "@/components/Insights/temp-flags";
import { IconLayoutWrapper } from "../IconLayoutWrapper";

export function NoResults() {
  const subheaderPrefix =
    "We couldn't find any results matching your search. Try adjusting your query";
  const subheader = SHOW_EXAMPLES_LINK
    ? `${subheaderPrefix} or browse our examples for inspiration.`
    : `${subheaderPrefix}.`;

  return (
    <IconLayoutWrapper
      action={
        SHOW_EXAMPLES_LINK ? (
          <Button
            appearance="outlined"
            disabled
            icon={<RiExternalLinkLine />}
            iconSide="left"
            kind="primary"
            label="See examples"
            size="medium"
          />
        ) : null
      }
      header="No results found"
      subheader={subheader}
    />
  );
}
