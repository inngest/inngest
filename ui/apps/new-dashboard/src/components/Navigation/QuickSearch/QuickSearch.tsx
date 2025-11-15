"use client";

import { useEffect, useState } from "react";
import { Button } from "@inngest/components/Button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@inngest/components/Tooltip/Tooltip";
import { RiSearchLine } from "@remixicon/react";

import { QuickSearchModal } from "./QuickSearchModal";

type Props = {
  collapsed: boolean;
  envSlug: string;
  envName: string;
};

export function QuickSearch({ collapsed, envSlug, envName }: Props) {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setIsOpen((open) => !open);
      }
    }

    document.addEventListener("keydown", onKeyDown);

    return () => {
      document.removeEventListener("keydown", onKeyDown);
    };
  }, []);

  return (
    <>
      {collapsed ? null : (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              appearance="outlined"
              aria-label="Search by ID"
              className="group/search overflow-hidden px-1.5"
              icon={<RiSearchLine className="-mr-1 group-hover/search:mr-0" />}
              iconSide="left"
              kind="secondary"
              onClick={() => setIsOpen(true)}
              size="small"
              label={
                <span className="hidden group-hover/search:block">Search</span>
              }
            />
          </TooltipTrigger>

          <TooltipContent
            className="w-32 rounded text-xs"
            side="bottom"
            sideOffset={2}
          >
            You can also use <span className="font-bold">⌘ K</span> or{" "}
            <span className="font-bold">Ctrl K</span> to search
          </TooltipContent>
        </Tooltip>
      )}

      <QuickSearchModal
        envSlug={envSlug}
        envName={envName}
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
      />
    </>
  );
}
