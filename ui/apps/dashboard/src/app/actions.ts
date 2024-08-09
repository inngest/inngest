'use server';

import { cookies } from 'next/headers';

export async function getNavCollapsed(): Promise<boolean | undefined> {
  const cookieStore = cookies();
  const collapsed = cookieStore.get('navCollapsed')?.value;
  return collapsed ? collapsed === 'true' : undefined;
}
