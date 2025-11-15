"use client";

import { useState } from "react";
import { Button } from "@inngest/components/Button/Button";
import { RiArchive2Line } from "@remixicon/react";

import { EnvironmentType } from "@/gql/graphql";
import type { Environment } from "@/utils/environments";
import { EnvironmentArchiveModal } from "./EnvironmentArchiveModal";

type Props = {
  env: Environment;
};

export function EnvArchiveButton({ env }: Props) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Need to store local state since the mutations don't invalidate the cache,
  // which means that the env prop won't update. We should fix cache
  // invalidation
  const [isArchived, setIsArchived] = useState(env.isArchived);

  return (
    <>
      <Button
        appearance="outlined"
        className={!isArchived ? "text-error" : undefined}
        icon={<RiArchive2Line />}
        kind="secondary"
        onClick={(e) => {
          e.preventDefault();
          setIsModalOpen(true);
        }}
        size="small"
        title={isArchived ? "Unarchive" : "Archive"}
      />

      <EnvironmentArchiveModal
        envID={env.id}
        isArchived={isArchived}
        isBranchEnv={env.type === EnvironmentType.BranchChild}
        isOpen={isModalOpen}
        onCancel={() => {
          setIsModalOpen(false);
        }}
        onSuccess={() => {
          setIsModalOpen(false);
          setIsArchived(!isArchived);
        }}
      />
    </>
  );
}
