"use client";

import { useRouter } from "next/navigation";
import { Button } from "@inngest/components/Button";

import { type Environment } from "@/utils/environments";
import { pathCreator } from "@/utils/urls";

type Props = {
  env: Pick<Environment, "slug">;
};

export function EnvViewButton({ env }: Props) {
  const router = useRouter();

  return (
    <Button
      appearance="outlined"
      kind="secondary"
      label="View"
      size="small"
      onClick={() => router.push(pathCreator.apps({ envSlug: env.slug }))}
    />
  );
}
