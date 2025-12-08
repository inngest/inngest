"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useClerk, useSignIn } from "@clerk/nextjs";

import LoadingIcon from "@/icons/LoadingIcon";

export default function ImpersonationClient({
  actorToken,
}: {
  actorToken: string;
}) {
  const { isLoaded, signIn } = useSignIn();
  const { setActive } = useClerk();
  const router = useRouter();

  useEffect(() => {
    if (!isLoaded) return;

    const performImpersonation = async () => {
      try {
        const { createdSessionId } = await signIn.create({
          strategy: "ticket",
          ticket: actorToken,
        });

        await setActive({ session: createdSessionId });
        router.push("/");
      } catch (err) {
        console.error("Impersonation failed:", JSON.stringify(err, null, 2));
        router.push("/");
      }
    };

    performImpersonation();
  }, [isLoaded, signIn, setActive, actorToken, router]);

  return (
    <div className="flex h-full w-full items-center justify-center gap-2">
      <LoadingIcon />
      <span>Signing you in...</span>
    </div>
  );
}
