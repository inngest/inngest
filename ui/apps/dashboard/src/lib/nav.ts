import { createServerFn } from '@tanstack/react-start';
import { getCookie, setCookie } from '@tanstack/react-start/server';

export const toggleCollapsed = createServerFn().handler(async () => {
  const collapsed = getCookie('navCollapsed') === 'true';
  setCookie('X-Nav-Collapsed', collapsed ? 'false' : 'true');
});

export const navCollapsed = createServerFn().handler(
  async () => getCookie('navCollapsed') === 'true',
);
