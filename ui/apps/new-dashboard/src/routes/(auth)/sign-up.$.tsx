import SplitView from "@/components/SignIn/SplitView";
import { SignUp } from "@clerk/tanstack-react-start";
import { createFileRoute } from "@tanstack/react-router";
import { createServerFn } from "@tanstack/react-start";
import { getCookie } from "@tanstack/react-start/server";

const getAnonymousId = createServerFn({ method: "GET" }).handler(async () => {
  const anonymousId = getCookie("inngest_anonymous_id");
  return anonymousId || null;
});

export const Route = createFileRoute("/(auth)/sign-up/$")({
  component: RouteComponent,
  loader: async () => {
    const anonymousId = await getAnonymousId();
    return { anonymousId };
  },
});

function RouteComponent() {
  const { anonymousId } = Route.useLoaderData();

  return (
    <SplitView>
      <div className="mx-auto my-8 mt-auto text-center">
        <SignUp
          unsafeMetadata={{
            ...(anonymousId && { anonymousID: anonymousId }),
          }}
          appearance={{
            elements: {
              footer: "bg-none",
            },
          }}
        />
      </div>
      <p className="text-subtle mt-auto text-center text-xs">
        By signing up, you agree to our{" "}
        <a
          className="text-link hover:underline"
          href="https://inngest.com/terms"
          target="_blank"
        >
          Terms of Service
        </a>{" "}
        and{" "}
        <a
          className="text-link hover:underline"
          href="https://inngest.com/privacy"
          target="_blank"
        >
          Privacy Policy
        </a>
        .
      </p>
    </SplitView>
  );
}
