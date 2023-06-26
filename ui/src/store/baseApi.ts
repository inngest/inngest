import { createApi } from "@reduxjs/toolkit/query/react";
import { graphqlRequestBaseQuery } from "@rtk-query/graphql-request-base-query";
import { GraphQLClient } from "graphql-request";

const hostname = process.env.NODE_ENV === 'development' ? "localhost:8288" : window.location.host;

export const client = new GraphQLClient(`http://${hostname}/v0/gql`);

export const api = createApi({
  baseQuery: graphqlRequestBaseQuery({ client }),
  endpoints: () => ({}),
});
