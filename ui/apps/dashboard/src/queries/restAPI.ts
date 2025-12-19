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
          return;
        }

        const headers = new Headers(request.headers);
        headers.set('Authorization', `Bearer ${sessionToken}`);

        return new Request(request, { headers });
      },
    ],
  },
});

export default restAPI;
