import {
  GraphQLClient,
  type RequestMiddleware,
  type ResponseMiddleware,
} from 'graphql-request';
import { notFound } from '@tanstack/react-router';

const decodeJWT = (token: string) => {
  try {
    const payload = token.split('.')[1];
    return JSON.parse(atob(payload));
  } catch {
    return null;
  }
};

const requestMiddleware: RequestMiddleware = async (request) => {
  //
  // Lazy import server-only modules to prevent them from being bundled for the client
  const { auth } = await import('@clerk/tanstack-react-start/server');
  const { getCookies } = await import('@tanstack/react-start/server');

  const { getToken, sessionId, userId } = await auth();
  let sessionToken = await getToken();

  //
  // Debug logging for auth issues
  console.log('[graphqlAPI] Auth state:', {
    hasToken: !!sessionToken,
    sessionId,
    userId,
    tokenLength: sessionToken?.length,
  });

  //
  // In serverless environments after login, tokens may be briefly stale.
  // Check if token expires very soon and retry once after a short delay.
  if (sessionToken) {
    const decoded = decodeJWT(sessionToken);
    const now = Date.now();
    const expiresIn = decoded?.exp ? decoded.exp * 1000 - now : null;
    const issuedAt = decoded?.iat ? decoded.iat * 1000 : null;

    console.log('[graphqlAPI] Token details:', {
      expiresIn: expiresIn ? `${Math.round(expiresIn / 1000)}s` : 'unknown',
      issuedAt: issuedAt ? new Date(issuedAt).toISOString() : 'unknown',
      isExpired: expiresIn !== null && expiresIn < 0,
      claims: decoded ? Object.keys(decoded) : [],
    });

    //
    // If token expires in < 5 seconds, it might be stale - wait and retry
    if (expiresIn !== null && expiresIn < 5000) {
      console.log('[graphqlAPI] Token expiring soon, retrying...');
      await new Promise((resolve) => setTimeout(resolve, 100));
      sessionToken = await getToken();
      const retryDecoded = decodeJWT(sessionToken || '');
      const retryExpiresIn = retryDecoded?.exp
        ? retryDecoded.exp * 1000 - Date.now()
        : null;
      console.log('[graphqlAPI] Retry token expiresIn:', retryExpiresIn);
    }
  } else {
    console.log('[graphqlAPI] No session token, using cookies');
  }

  let headers = request.headers;
  if (sessionToken) {
    headers = {
      ...headers,
      Authorization: `Bearer ${sessionToken}`,
    };
  } else {
    //
    // Need to forward the `Cookie` header for non-Clerk users.
    const allCookies = getCookies();
    const cookieString = Object.entries(allCookies)
      .map(([name, value]) => `${name}=${value}`)
      .join('; ');

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
  if (response instanceof Error && response.message.includes('not found')) {
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

export default graphqlAPI;
