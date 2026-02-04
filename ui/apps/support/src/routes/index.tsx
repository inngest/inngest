import { createFileRoute } from "@tanstack/react-router";
import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import { createServerFn } from "@tanstack/react-start";
import { usePaginationUI } from "@inngest/components/Pagination";
// import { Filters } from "@/components/Support/Filters";
import { Button } from "@inngest/components/Button";
import type { TicketSummary } from "@/data/plain";
import { getTicketsByEmail } from "@/data/plain";
import { TicketCard } from "@/components/Support/TicketCard";
import { CommunityChannels } from "@/components/Support/CommunityChannels";

const getAuthStatusAndTickets = createServerFn({ method: "GET" }).handler(
  async () => {
    const { isAuthenticated, userId } = await auth();

    // Only fetch user email and tickets if authenticated
    let userEmail: string | undefined = undefined;
    let tickets: Array<TicketSummary> = [];

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

  // Paginate tickets with 8 per page
  const { currentPageData, BoundPagination } = usePaginationUI({
    data: tickets,
    id: "tickets",
    pageSize: 8,
  });

  if (!isAuthenticated || !userEmail) {
    return (
      <div className="flex flex-col min-h-screen items-center justify-center bg-canvasBase">
        <div className="text-center">
          <h1 className="text-basis mb-2 text-2xl font-bold">
            Please sign in to view your tickets
          </h1>
          <p className="text-muted">
            You need to be authenticated to access the support portal.
          </p>
          <div className="flex justify-center mt-4">
            <Button
              kind="primary"
              appearance="outlined"
              label="Sign in"
              href="/sign-in"
            />
          </div>
        </div>
        <CommunityChannels />
      </div>
    );
  }

  return (
    <div className="mx-auto w-full max-w-5xl py-6">
      {/* Filters */}
      {/* <Filters /> */}

      {/* Ticket List */}
      <div className="flex w-full flex-col gap-4 py-4">
        <div className="text-basis flex w-full items-center justify-between leading-none">
          <div className="flex flex-col justify-center">
            <p className="leading-4 whitespace-nowrap">My tickets</p>
          </div>
          <div className="flex flex-col justify-center">
            <p className="leading-4 whitespace-nowrap">
              {tickets.length} {tickets.length === 1 ? "ticket" : "tickets"}
            </p>
          </div>
        </div>

        {tickets.length === 0 ? (
          <div className="border-muted bg-canvasSubtle flex flex-col items-center justify-center rounded-lg border p-12 text-center">
            <p className="text-basis mb-1 text-lg font-medium">
              No support tickets found
            </p>
            <p className="text-muted">
              Create a new ticket to get help from our support team.
            </p>
          </div>
        ) : (
          <>
            {currentPageData.map((ticket) => (
              <TicketCard key={ticket.id} ticket={ticket} />
            ))}

            {/* Pagination */}
            {tickets.length > 8 && (
              <div className="flex w-full flex-col items-center justify-center px-0 py-3">
                <BoundPagination />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
