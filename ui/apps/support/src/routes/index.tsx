import { createFileRoute, Link } from "@tanstack/react-router";
import { ProfileMenu } from "@/components/Navigation/ProfileMenu";
import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import { createServerFn } from "@tanstack/react-start";
import { useClerk } from "@clerk/tanstack-react-start";
import { getTicketsByEmail, type TicketSummary } from "@/data/plain";
import { StatusBadge, PriorityBadge } from "@/components/Support/TicketBadges";
import { usePaginationUI } from "@inngest/components/Pagination";

const getAuthStatusAndTickets = createServerFn({ method: "GET" }).handler(
  async () => {
    const { isAuthenticated, userId } = await auth();

    // Only fetch user email and tickets if authenticated
    let userEmail: string | undefined = undefined;
    let tickets: TicketSummary[] = [];

    if (isAuthenticated && userId) {
      try {
        const user = await clerkClient().users.getUser(userId);
        userEmail = user.emailAddresses[0]?.emailAddress;

        // Fetch tickets using the user's email
        if (userEmail) {
          tickets = await getTicketsByEmail({ data: { email: userEmail } });
        }
      } catch (error) {
        // If user fetch fails, user will see sign-in
        console.error("Failed to fetch user or tickets:", error);
      }
    }

    return {
      isAuthenticated,
      userEmail,
      tickets,
    };
  },
);

export const Route = createFileRoute("/")({
  component: Home,
  loader: async () => {
    return await getAuthStatusAndTickets();
  },
});

function Home() {
  const { isAuthenticated, userEmail, tickets } = Route.useLoaderData();
  const { signOut, session } = useClerk();

  // Paginate tickets with 4 per page
  const { currentPageData, BoundPagination } = usePaginationUI({
    data: tickets,
    id: "tickets",
    pageSize: 8,
  });

  const handleSignOut = async () => {
    await signOut({
      sessionId: session?.id,
      redirectUrl: "/sign-in/$",
    });
  };

  return (
    <div className="min-h-screen bg-canvasBase">
      <div className="mx-auto max-w-6xl px-6 py-8">
        <div className="mb-8">
          <h1 className="text-basis mb-2 text-3xl font-bold">
            Inngest Support Portal
          </h1>
          <p className="text-muted text-base">
            Manage your support tickets and get help from our team
          </p>
          <div className="mt-4">
            {isAuthenticated && userEmail ? (
              <div className="flex items-center gap-3">
                <span className="text-muted text-sm">
                  Logged in as:{" "}
                  <span className="font-medium text-basis">{userEmail}</span>
                </span>
                <span className="text-subtle">•</span>
                <button
                  onClick={handleSignOut}
                  className="text-muted hover:text-basis text-sm underline transition-colors"
                >
                  Sign Out
                </button>
              </div>
            ) : (
              <ProfileMenu isAuthenticated={false}>
                <div></div>
              </ProfileMenu>
            )}
          </div>
        </div>

        {isAuthenticated && userEmail && (
          <div className="mt-8">
            <div className="mb-6 flex items-center justify-between">
              <h2 className="text-basis text-xl font-semibold">
                Your Support Tickets
              </h2>
              {tickets.length > 0 && (
                <span className="text-muted text-sm">
                  {tickets.length} {tickets.length === 1 ? "ticket" : "tickets"}
                </span>
              )}
            </div>
            {tickets.length === 0 ? (
              <div className="bg-canvasSubtle border-subtle rounded-xl border p-12 text-center">
                <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-canvasMuted">
                  <svg
                    className="text-muted h-8 w-8"
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
                </div>
                <p className="text-basis mb-1 text-lg font-medium">
                  No support tickets found
                </p>
                <p className="text-muted text-sm">
                  Create a new ticket to get help from our support team.
                </p>
              </div>
            ) : (
              <>
                <div className="space-y-3">
                  {currentPageData.map((ticket) => (
                    <Link
                      key={ticket.id}
                      to="/case/$ticketId"
                      params={{ ticketId: ticket.id }}
                      className="bg-canvasBase border-subtle hover:border-basis hover:shadow-sm group block rounded-lg border p-5 transition-all duration-200"
                    >
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1 min-w-0">
                          <h3 className="text-basis mb-3 text-base font-semibold group-hover:text-link transition-colors line-clamp-2">
                            {ticket.title}
                          </h3>
                          <div className="mb-3 flex flex-wrap items-center gap-2">
                            <StatusBadge status={ticket.status} size="sm" />
                            <PriorityBadge
                              priority={ticket.priority}
                              size="sm"
                              showLabel={false}
                            />
                          </div>
                          <div className="flex items-center gap-4 text-xs text-muted">
                            <span>
                              Created:{" "}
                              {new Date(ticket.createdAt).toLocaleDateString()}
                            </span>
                            <span>•</span>
                            <span>
                              Updated:{" "}
                              {new Date(ticket.updatedAt).toLocaleDateString()}
                            </span>
                          </div>
                        </div>
                        <div className="text-muted shrink-0">
                          <svg
                            className="h-5 w-5 transition-transform group-hover:translate-x-1"
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
                        </div>
                      </div>
                    </Link>
                  ))}
                </div>
                {tickets.length > 4 && (
                  <div className="mt-6">
                    <BoundPagination />
                  </div>
                )}
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
