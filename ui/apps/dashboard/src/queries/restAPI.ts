import { auth } from '@clerk/nextjs/server';
import ky from 'ky';

export { HTTPError } from 'ky';

const restAPI = ky.create({
  prefixUrl: `${process.env.NEXT_PUBLIC_API_URL}/v1`,
  hooks: {
    beforeRequest: [
      async (request) => {
        const { getToken } = auth();
        const sessionToken = await getToken();

        // TODO: Does this need to be changed for Vercel Marketplace? Vercel
        // Marketplace users don't auth with Clerk.
        if (!sessionToken) return;

        request.headers.set('Authorization', `Bearer ${sessionToken}`);
      },
    ],
  },
});

export default restAPI;
