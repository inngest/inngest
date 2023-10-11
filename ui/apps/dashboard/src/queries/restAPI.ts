import { auth } from '@clerk/nextjs';
import ky from 'ky';

export { HTTPError } from 'ky';

const restAPI = ky.create({
  prefixUrl: `${process.env.NEXT_PUBLIC_API_URL}/v1`,
  hooks: {
    beforeRequest: [
      async (request) => {
        const { getToken } = auth();
        const sessionToken = await getToken();
        if (!sessionToken) return;
        request.headers.set('Authorization', `Bearer ${sessionToken}`);
      },
    ],
  },
});

export default restAPI;
