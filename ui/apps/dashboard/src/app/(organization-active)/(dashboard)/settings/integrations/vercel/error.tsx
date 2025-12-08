"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@inngest/components/Button";
import * as Sentry from "@sentry/nextjs";

import { FatalError } from "@/components/FatalError";
import { pathCreator } from "@/utils/urls";

type VercelIntegrationErrorProps = {
  error: Error & { digest?: string };
  reset: () => void;
};

export default function VercelIntegrationError({
  error,
  reset,
}: VercelIntegrationErrorProps) {
  const router = useRouter();

  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <>
      <FatalError error={error} reset={reset} />
      <div className="m-auto mt-4 flex w-fit flex-col items-center gap-2 text-center">
        <p className="text-sm text-slate-600 dark:text-slate-400">
          If you continue to have issues loading this page, try reconnecting
          your Inngest account to Vercel.
        </p>
        <Button
          onClick={() => router.push(pathCreator.vercelSetup())}
          kind="secondary"
          appearance="outlined"
          label="Reconnect Account"
        />
      </div>
    </>
  );
}
