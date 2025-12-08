"use client";

import { Alert } from "@inngest/components/Alert";

import { useEnvironment } from "@/components/Environments/environment-context";
import { EnvironmentType } from "@/utils/environments";
import useManagePageTerminology from "./useManagePageTerminology";

export default function EventKeysPage() {
  const currentContent = useManagePageTerminology();
  const environment = useEnvironment();
  const shouldShowAlert =
    currentContent?.param === "keys" &&
    environment.type === EnvironmentType.BranchParent;

  return (
    <div className="flex h-full w-full flex-col">
      {shouldShowAlert && (
        <Alert
          className="flex items-center rounded-none text-sm"
          severity="info"
        >
          Event keys are shared for all branch environments
        </Alert>
      )}
      <div className="flex flex-1 items-center justify-center">
        <h2 className="text-subtle text-sm font-semibold">
          {"Select a " + currentContent?.type + " on the left."}
        </h2>
      </div>
    </div>
  );
}
