import { createFileRoute } from "@tanstack/react-router";
import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import { createServerFn } from "@tanstack/react-start";
import { usePaginationUI } from "@inngest/components/Pagination";
// import { Filters } from "@/components/Support/Filters";
import { Button } from "@inngest/components/Button";
import type { TicketSummary } from "@/data/plain";
import { getTicketsByEmail } from "@/data/plain";
import { TicketCard } from "@/components/Support/TicketCard";
import { Link } from "@inngest/components/Link";
import { RiGithubFill } from "@remixicon/react";

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
        <div className="mt-8 p-4 border rounded flex flex-col gap-4 max-w-xl">
          <h2 className="text-basis text-lg font-bold">Community</h2>
          <p>
            Chat with other developers and the Inngest team in our{" "}
            <Link
              target="_blank"
              href="https://www.inngest.com/discord"
              className="inline-flex"
              size="medium"
            >
              Discord community
            </Link>
            . Search for topics and questions in our{" "}
            <Link
              href="https://discord.com/channels/842170679536517141/1051516534029291581"
              className="inline-flex"
              target="_blank"
              size="medium"
            >
              #help-forum
            </Link>{" "}
            channel or submit your own question.
          </p>
          <Button
            kind="primary"
            href="https://www.inngest.com/discord"
            target="_blank"
            label="Join our Discord"
          />
          <h2 className="mt-4 text-basis text-lg font-bold">Open source</h2>
          <p>File an issue in our open source repos on Github:</p>
          <div>
            <p className="mb-2 text-sm font-medium">Inngest CLI + Dev Server</p>
            <Button
              appearance="outlined"
              kind="secondary"
              href="https://github.com/inngest/inngest/issues"
              label="inngest/inngest"
              icon={<RiGithubFill />}
              className="justify-start"
              iconSide="left"
            />
          </div>
          <div>
            <p className="mb-2 text-sm font-medium">SDKs</p>
            <div className="flex flex-row gap-2">
              <Button
                appearance="outlined"
                kind="secondary"
                href="https://github.com/inngest/inngest-js/issues"
                label="inngest/inngest-js"
                icon={<RiGithubFill />}
                iconSide="left"
              />
              <Button
                appearance="outlined"
                kind="secondary"
                href="https://github.com/inngest/inngest-py/issues"
                label="inngest/inngest-py"
                icon={<RiGithubFill />}
                iconSide="left"
              />
              <Button
                appearance="outlined"
                kind="secondary"
                href="https://github.com/inngest/inngestgo/issues"
                label="inngest/inngestgo"
                icon={<RiGithubFill />}
                iconSide="left"
              />
            </div>
          </div>
        </div>
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
