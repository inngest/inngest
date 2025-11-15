import { getCookies } from "@tanstack/react-start/server";

import {
  GraphQLClient,
  type RequestMiddleware,
  type ResponseMiddleware,
} from "graphql-request";
import { notFound } from "@tanstack/react-router";
import { auth } from "@clerk/tanstack-react-start/server";

const requestMiddleware: RequestMiddleware = async (request) => {
  const { getToken } = await auth();
  const sessionToken = await getToken();

  let headers = request.headers;
  if (sessionToken) {
    headers = {
      ...headers,
      Authorization: `Bearer ${sessionToken}`,
    };
  } else {
    // Need to forward the `Cookie` header for non-Clerk users.
    // TANSTACK TODO: what is this? do we still need it
    const allCookies = getCookies();
    const cookieString = Object.entries(allCookies)
      .map(([name, value]) => `${name}=${value}`)
      .join("; ");

    headers = {
      ...headers,
      Cookie: cookieString,
    };
  }

  return {
    ...request,
    headers,
  };
};

/**
 * Throws the not found error when a requested resource wasn't found, which can be
 * handled gracefully by an enclosing `not-found` file.
 */
const throwNotFoundError: ResponseMiddleware = (response) => {
  if (response instanceof Error && response.message.includes("not found")) {
    notFound();
  }
};

export const graphqlAPI = new GraphQLClient(
  `${import.meta.env.VITE_API_URL}/gql`,
  {
    requestMiddleware,
    responseMiddleware: throwNotFoundError,
  },
);
