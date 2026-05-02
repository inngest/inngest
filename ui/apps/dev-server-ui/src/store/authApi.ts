import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

const baseURL = import.meta.env.VITE_PUBLIC_API_BASE_URL
  ? new URL('/', import.meta.env.VITE_PUBLIC_API_BASE_URL)
  : '/';

export interface AuthStatus {
  authRequired: boolean;
  authenticated: boolean;
}

interface LoginRequest {
  email: string;
  password: string;
}

export const authApi = createApi({
  reducerPath: 'authApi',
  baseQuery: fetchBaseQuery({
    baseUrl: baseURL.toString(),
    credentials: 'include',
  }),
  endpoints: (builder) => ({
    authStatus: builder.query<AuthStatus, void>({
      query: () => '/auth/status',
    }),
    login: builder.mutation<{ ok: boolean }, LoginRequest>({
      query: (body) => ({
        url: '/auth/login',
        method: 'POST',
        body,
      }),
    }),
    logout: builder.mutation<{ ok: boolean }, void>({
      query: () => ({
        url: '/auth/logout',
        method: 'POST',
      }),
    }),
  }),
});

export const { useAuthStatusQuery, useLoginMutation, useLogoutMutation } =
  authApi;
