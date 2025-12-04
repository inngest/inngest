import { createFileRoute, Link } from "@tanstack/react-router";
import { ProfileMenu } from "@/components/Navigation/ProfileMenu";
import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import { createServerFn } from "@tanstack/react-start";
import { useClerk } from "@clerk/tanstack-react-start";
import { getTicketsByEmail, type TicketSummary } from "@/data/plain";

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

  const handleSignOut = async () => {
    await signOut({
      sessionId: session?.id,
      redirectUrl: "/sign-in/$",
    });
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case "todo":
        return "bg-yellow-100 text-yellow-800";
      case "done":
        return "bg-green-100 text-green-800";
      case "snoozed":
        return "bg-blue-100 text-blue-800";
      default:
        return "bg-gray-100 text-gray-800";
    }
  };

  const getPriorityColor = (priority: string) => {
    const priorityStr = priority ? String(priority).toLowerCase() : "";
    switch (priorityStr) {
      case "urgent":
        return "text-red-600";
      case "high":
        return "text-orange-600";
      case "normal":
        return "text-blue-600";
      case "low":
        return "text-gray-600";
      default:
        return "text-gray-600";
    }
  };

  return (
    <div className="px-6 pt-4">
      <div className="mb-6">
        <h1 className="text-basis text-2xl font-semibold">
          Inngest Support Portal
        </h1>
        <div className="mt-2">
          {isAuthenticated && userEmail ? (
            <div className="flex items-center gap-2">
              <span className="text-muted text-sm">
                Logged in as: {userEmail}
              </span>
              <button
                onClick={handleSignOut}
                className="text-muted hover:text-basis text-sm underline"
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
          <h2 className="text-basis mb-4 text-xl font-semibold">
            Your Support Tickets
          </h2>
          {tickets.length === 0 ? (
            <div className="bg-canvasSubtle text-muted rounded-lg border border-subtle p-8 text-center">
              <p>No support tickets found.</p>
              <p className="mt-2 text-sm">
                Create a new ticket to get help from our support team.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {tickets.map((ticket) => (
                <Link
                  key={ticket.id}
                  to="/case/$ticketId"
                  params={{ ticketId: ticket.id }}
                  className="bg-canvasBase border-subtle hover:border-basis block rounded-lg border p-4 transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <h3 className="text-basis font-medium">{ticket.title}</h3>
                      <div className="mt-2 flex items-center gap-3">
                        <span
                          className={`rounded-full px-2 py-1 text-xs font-medium ${getStatusColor(
                            ticket.status,
                          )}`}
                        >
                          {ticket.status}
                        </span>
                        <span
                          className={`text-xs font-medium ${getPriorityColor(
                            ticket.priority,
                          )}`}
                        >
                          {ticket.priority} priority
                        </span>
                      </div>
                      <div className="text-muted mt-2 text-xs">
                        Updated:{" "}
                        {new Date(ticket.updatedAt).toLocaleDateString()}
                      </div>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
