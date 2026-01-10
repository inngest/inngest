import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useServerFn } from "@tanstack/react-start";
import { Header } from "@inngest/components/Header/Header";
import { getProfileDisplay } from "@/data/profile";
import { envQueryOptions } from "@/data/envs";

export const Route = createFileRoute("/_authed/support/")({
  component: HomeComponent,
  loader: async ({ context }) => {
    const envs = await context.queryClient.ensureQueryData(
      envQueryOptions("production"),
    );

    return {
      envs,
    };
  },
});

function HomeComponent() {
  const { envs } = Route.useLoaderData();
  const getProfile = useServerFn(getProfileDisplay);

  const { data } = useQuery({
    queryKey: ["profile"],
    queryFn: () => getProfile(),
  });

  return (
    <>
      <Header breadcrumb={[{ text: "Support" }]} />
      <div className="min-h-screen bg-canvasBase">
        <div className="mx-auto max-w-6xl px-6 py-8">
          <div className="mb-8">
            <h1 className="text-basis mb-2 text-3xl font-bold">
              Support Center
            </h1>
            <p className="text-muted text-base">
              Get help, find resources, and manage your support tickets
            </p>
          </div>

          <div className="grid gap-6 md:grid-cols-2">
            {/* Quick Actions Card */}
            <div className="bg-canvasSubtle border-subtle rounded-xl border p-6">
              <h2 className="text-basis mb-4 text-lg font-semibold">
                Quick Actions
              </h2>
              <div className="space-y-3">
                <a
                  href="/"
                  className="text-link hover:text-linkHover flex items-center gap-3 text-sm font-medium transition-colors"
                >
                  <svg
                    className="h-5 w-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"
                    />
                  </svg>
                  View all tickets
                </a>
                <a
                  href="https://www.inngest.com/docs"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-link hover:text-linkHover flex items-center gap-3 text-sm font-medium transition-colors"
                >
                  <svg
                    className="h-5 w-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"
                    />
                  </svg>
                  Documentation
                </a>
                <a
                  href="https://status.inngest.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-link hover:text-linkHover flex items-center gap-3 text-sm font-medium transition-colors"
                >
                  <svg
                    className="h-5 w-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  System Status
                </a>
              </div>
            </div>

            {/* Account Info Card */}
            <div className="bg-canvasSubtle border-subtle rounded-xl border p-6">
              <h2 className="text-basis mb-4 text-lg font-semibold">
                Account Information
              </h2>
              <div className="space-y-3 text-sm">
                {data?.displayName && (
                  <div>
                    <span className="text-muted">Display Name:</span>{" "}
                    <span className="text-basis font-medium">
                      {data.displayName}
                    </span>
                  </div>
                )}
                {envs.envBySlug?.name && (
                  <div>
                    <span className="text-muted">Environment:</span>{" "}
                    <span className="text-basis font-medium">
                      {envs.envBySlug.name}
                    </span>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Help Section */}
          <div className="mt-8 bg-canvasSubtle border-subtle rounded-xl border p-6">
            <h2 className="text-basis mb-4 text-lg font-semibold">
              Need Help?
            </h2>
            <p className="text-muted mb-4 text-sm">
              Our support team is here to help. If you have a question or need
              assistance, please create a support ticket and we'll get back to
              you as soon as possible.
            </p>
            <a
              href="/"
              className="text-link hover:text-linkHover inline-flex items-center gap-2 text-sm font-medium transition-colors"
            >
              View your tickets
              <svg
                className="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 5l7 7-7 7"
                />
              </svg>
            </a>
          </div>
        </div>
      </div>
    </>
  );
}
