import { auth, clerkClient } from "@clerk/tanstack-react-start/server";
import { redirect } from "@tanstack/react-router";
import { createMiddleware, createServerFn } from "@tanstack/react-start";

/**
 * Get the authenticated user's email address from Clerk.
 * Returns null if the user is not authenticated or has no email.
 */
export async function getAuthenticatedUserEmail(): Promise<string | null> {
  const { isAuthenticated, userId } = await auth();
  if (!isAuthenticated || !userId) return null;
  const user = await clerkClient().users.getUser(userId);
  return user.emailAddresses[0]?.emailAddress ?? null;
}

/**
 * Middleware that requires an authenticated user and provides their email
 * via context. Use with `.middleware([authMiddleware])` on server functions.
 */
export const authMiddleware = createMiddleware().server(async ({ next }) => {
  const email = await getAuthenticatedUserEmail();
  if (!email) {
    throw new Error("Not authenticated");
  }
  return next({ context: { userEmail: email } });
});

export const fetchClerkAuth = createServerFn({ method: "GET" }).handler(
  async () => {
    const { isAuthenticated, userId, getToken } = await auth();

    if (!isAuthenticated) {
      throw redirect({
        to: "/sign-in/$",
      });
    }

    const token = await getToken();
    return {
      userId,
      token,
    };
  },
);

export const fetchUserProfile = createServerFn({ method: "GET" }).handler(
  async () => {
    const { isAuthenticated, userId, getToken } = await auth();

    if (!isAuthenticated) {
      throw redirect({
        to: "/sign-in/$",
      });
    }

    const token = await getToken();
    return {
      userId,
      token,
    };
  },
);
