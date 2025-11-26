import SplitView from "@/components/SignIn/SplitView";
import { SignUp } from "@clerk/tanstack-react-start";
import { createFileRoute } from "@tanstack/react-router";

//
// TANSTACK TODO: Add anonymous ID to the sign up form
export const Route = createFileRoute("/sign-up/$")({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <SplitView>
      <div className="mx-auto my-8 mt-auto text-center">
        <SignUp
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
