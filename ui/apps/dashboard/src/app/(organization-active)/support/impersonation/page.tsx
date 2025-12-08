import { redirect } from "next/navigation";
import { auth } from "@clerk/nextjs/server";

import ImpersonationClient from "./impersonationClient";

export default async function ImpersonateUsers({
  searchParams,
}: {
  searchParams: { [key: string]: string | string[] | undefined };
}) {
  const user = auth();
  const userId = searchParams["user_id"];

  if (!user.userId || !userId) {
    console.log("Missing user or userId");
    return redirect("/");
  }

  const INNGEST_ORG_ID = process.env.CLERK_INNGEST_ORG_ID;

  if (!INNGEST_ORG_ID) {
    console.log("Missing CLERK_INNGEST_ORG_ID env variable");
    return redirect("/");
  }

  if (user.orgId !== INNGEST_ORG_ID) {
    console.log("User is not in INNGEST_ORG_ID");
    return redirect("/");
  }
  const actorId = user.userId;

  const params = JSON.stringify({
    user_id: userId,
    actor: {
      sub: actorId,
    },
  });

  if (!process.env.CLERK_SECRET_KEY) {
    console.log("Missing CLERK_SECRET_KEY env variable");
    return redirect("/");
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
      body: params,
      cache: "no-store",
    });
  } catch (e) {
    return { ok: false, error: "Network error while contacting Clerk" };
  }

  if (!res.ok) {
    return { ok: false, error: "Failed to generate actor token" };
  }
  const data = await res.json();

  return <ImpersonationClient actorToken={data.token} />;
}
