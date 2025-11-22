import { createFileRoute } from "@tanstack/react-router";
import { SignIn } from "@clerk/tanstack-react-start";
import SplitView from "@/components/SignIn/SplitView";
import { Alert } from "@inngest/components/Alert/NewAlert";
import SignInRedirectErrors, {
  hasErrorMessage,
} from "@/components/SignIn/Errors";

type SignInSearchParams = {
  redirect_url?: string;
  error?: string;
};

export const Route = createFileRoute("/sign-in/$")({
  component: RouteComponent,
  validateSearch: (search: Record<string, unknown>): SignInSearchParams => {
    return {
      redirect_url: search?.redirect_url as string | undefined,
      error: search?.error as string | undefined,
    };
  },
});

function RouteComponent() {
  const { error } = Route.useSearch();

  return (
    <SplitView>
      <div className="mx-auto my-auto text-center">
        <SignIn
          appearance={{
            elements: {
              footer: "bg-none",
            },
          }}
        />
        {typeof error === "string" && (
          <Alert severity="error" className="mx-auto max-w-xs">
            <p className="text-balance">
              {hasErrorMessage(error) ? SignInRedirectErrors[error] : error}
            </p>
            <p className="mt-2">
              <Alert.Link
                size="medium"
                severity="error"
                href="/support"
                className="inline underline"
              >
                Contact support
              </Alert.Link>{" "}
              if this problem persists.
            </p>
          </Alert>
        )}
      </div>
    </SplitView>
  );
}
