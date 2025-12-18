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

        console.log('rest api user id and session token', userId, sessionToken);

        // TODO: Does this need to be changed for Vercel Marketplace? Vercel
        // Marketplace users don't auth with Clerk.
        if (!sessionToken) return;

        request.headers.set('Authorization', `Bearer ${sessionToken}`);
      },
    ],
  },
});

export default restAPI;
