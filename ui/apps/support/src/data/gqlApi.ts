import { GraphQLClient } from "graphql-request";
import type { RequestMiddleware } from "graphql-request";

export const API = `${import.meta.env.VITE_API_URL}/gql`;

if (!API) {
  console.error(
    "VITE_API_URL is required to use the Inngest graphql api",
    import.meta.env,
  );
}

export const getAuthHeaders = async () => {
  const { auth } = await import("@clerk/tanstack-react-start/server");

  const { getToken } = await auth();
  const token = await getToken();
  return {
    authorization: `Bearer ${token}`,
  };
};

const requestMiddleware: RequestMiddleware = async (request) => {
  return {
    ...request,
    headers: {
      ...request.headers,
      ...(await getAuthHeaders()),
    },
  };
};

export const inngestGQLAPI = new GraphQLClient(API, {
  requestMiddleware,
});
