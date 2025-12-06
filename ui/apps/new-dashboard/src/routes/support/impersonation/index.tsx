import ImpersonationClient from "@/components/Impersonation/Client";
import { auth } from "@clerk/tanstack-react-start/server";
import {
  createFileRoute,
  redirect,
  useLoaderData,
  useSearch,
} from "@tanstack/react-router";

type ImpersonationLoaderData = {
  actorToken?: string;
};
export const Route = createFileRoute("/support/impersonation/")({
  component: ImpersonationComponent,
  validateSearch: (search: Record<string, unknown>) => {
    return {
      userId: search.userId as string,
    };
  },
  loader: async ({ params: { userId } }: { params: { userId: string } }) => {
    const user = await auth();

    if (!user.userId || !userId) {
      console.log("Missing user or userId");
      redirect({ to: "/", throw: true });
    }

    const INNGEST_ORG_ID = import.meta.env.CLERK_INNGEST_ORG_ID;

    if (!INNGEST_ORG_ID) {
      console.log("Missing CLERK_INNGEST_ORG_ID env variable");
      redirect({ to: "/", throw: true });
    }

    if (user.orgId !== INNGEST_ORG_ID) {
      console.log("User is not in INNGEST_ORG_ID");
      redirect({ to: "/", throw: true });
    }
    const actorId = user.userId;

    const body = JSON.stringify({
      user_id: userId,
      actor: {
        sub: actorId,
      },
    });

    if (!process.env.CLERK_SECRET_KEY) {
      console.log("Missing CLERK_SECRET_KEY env variable");
      redirect({ to: "/", throw: true });
    }
    console.log("ACTOR: ", user.userId);
    console.log("USER: ", actorId);

    let res: Response;
    try {
      res = await fetch("https://api.clerk.com/v1/actor_tokens", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${process.env.CLERK_SECRET_KEY}`,
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body,
        cache: "no-store",
      });
    } catch (e) {
      return { ok: false, error: "Network error while contacting Clerk" };
    }

    if (!res.ok) {
      return { ok: false, error: "Failed to generate actor token" };
    }
    const data = await res.json();
    return { actorToken: data.token };
  },
});

function ImpersonationComponent() {
  const data = useLoaderData({ strict: false });
  return (
    data?.actorToken && <ImpersonationClient actorToken={data.actorToken} />
  );
}
