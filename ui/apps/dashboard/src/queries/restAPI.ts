import ky from 'ky';

export { HTTPError } from 'ky';

const restAPI = ky.create({
  prefixUrl: `${import.meta.env.VITE_API_URL}/v1`,
  hooks: {
    beforeRequest: [
      async (request) => {
        //
        // Lazy import server-only modules to prevent them from being bundled for the client
        const { auth } = await import('@clerk/tanstack-react-start/server');

        const { getToken } = await auth();
        const sessionToken = await getToken();

        //
        // TODO: Does this need to be changed for Vercel Marketplace? Vercel
        // Marketplace users don't auth with Clerk.
        if (!sessionToken) {
          console.log('No session token, skipping auth header');
          return;
        }

        //
        // Create new headers object with Authorization header (matching graphqlAPI pattern)
        const headers = new Headers(request.headers);
        headers.set('Authorization', `Bearer ${sessionToken}`);

        const newRequest = new Request(request, { headers });

        //
        // Debug: Log the request being sent with headers
        console.log('Sending request:', {
          url: newRequest.url,
          method: newRequest.method,
          hasAuthHeader: headers.has('authorization'),
          tokenPrefix: sessionToken.substring(0, 50) + '...',
        });

        return newRequest;
      },
    ],
  },
});

export default restAPI;
