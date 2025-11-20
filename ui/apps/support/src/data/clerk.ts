import { auth } from "@clerk/tanstack-react-start/server";
import { redirect } from "@tanstack/react-router";
import { createServerFn } from "@tanstack/react-start";

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
