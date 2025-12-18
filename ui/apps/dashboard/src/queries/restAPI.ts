import { auth } from '@clerk/tanstack-react-start/server';
import ky from 'ky';

export { HTTPError } from 'ky';

const restAPI = ky.create({
  prefixUrl: `${import.meta.env.VITE_API_URL}/v1`,
  hooks: {
    beforeRequest: [
      async (request) => {
        const { getToken, userId } = await auth();
        const sessionToken = await getToken();

        //
        // TODO: Does this need to be changed for Vercel Marketplace? Vercel
        // Marketplace users don't auth with Clerk.
        if (!sessionToken) {
          console.log('No session token, skipping auth header');
          return;
        }

        const headers = new Headers(request.headers);
        headers.set('Authorization', `Bearer ${sessionToken}`);

        const newRequest = new Request(request, { headers });

        //
        // Debug: Log the request being sent with headers
        console.log('Sending request:', {
          url: newRequest.url,
          method: newRequest.method,
          headers: Object.fromEntries(newRequest.headers.entries()),
        });

        return newRequest;
      },
    ],
  },
});

export default restAPI;
