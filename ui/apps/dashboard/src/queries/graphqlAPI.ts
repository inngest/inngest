import { cookies } from 'next/headers';
import { notFound } from 'next/navigation';
import { auth } from '@clerk/nextjs/server';
import { GraphQLClient, type RequestMiddleware, type ResponseMiddleware } from 'graphql-request';

import 'server-only';

const requestMiddleware: RequestMiddleware = async (request) => {
  const { getToken } = auth();
  const sessionToken = await getToken();

  let headers = request.headers;
  if (sessionToken) {
    headers = {
      ...headers,
      Authorization: `Bearer ${sessionToken}`,
    };
  } else {
    // Need to forward the `Cookie` header for non-Clerk users.

    const allCookies = cookies()
      .getAll()
      .map((c) => `${c.name}=${c.value}`)
      .join('; ');

    headers = {
      ...headers,
      Cookie: allCookies,
    };
  }

  return {
    ...request,
    headers,
  };
};

/**
 * Throws the `NEXT_NOT_FOUND` error when a requested resource wasn't found, which can be
 * handled gracefully by an enclosing `not-found` file.
 *
 * @see {@link https://beta.nextjs.org/docs/api-reference/notfound#notfound}
 */
const throwNotFoundError: ResponseMiddleware = (response) => {
  if (response instanceof Error && response.message.includes('not found')) {
    notFound();
  }
};

const graphqlAPI = new GraphQLClient(`${process.env.NEXT_PUBLIC_API_URL}/gql`, {
  requestMiddleware,
  responseMiddleware: throwNotFoundError,
  fetch, // use global fetch for Vercel Edge runtime
});

export default graphqlAPI;
