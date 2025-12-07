import { useEffect } from "react";
import { useClerk, useSignIn } from "@clerk/tanstack-react-start";

import LoadingIcon from "@/components/Icons/LoadingIcon";
import { useNavigate } from "@tanstack/react-router";

export default function ImpersonationClient({
  actorToken,
}: {
  actorToken: string;
}) {
  const { isLoaded, signIn } = useSignIn();
  const { setActive } = useClerk();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoaded) return;

    const performImpersonation = async () => {
      try {
        const { createdSessionId } = await signIn.create({
          strategy: "ticket",
          ticket: actorToken,
        });

        await setActive({ session: createdSessionId });
        navigate({ to: "/" });
      } catch (err) {
        console.error("Impersonation failed:", JSON.stringify(err, null, 2));
        navigate({ to: "/" });
      }
    };

    performImpersonation();
  }, [isLoaded, signIn, setActive, actorToken, navigate]);

  return (
    <div className="flex h-full w-full items-center justify-center gap-2">
      <LoadingIcon />
      <span>Signing you in...</span>
    </div>
  );
}
