'use server';

import { cookies } from 'next/headers';

export async function getNavCollapsed() {
  const cookieStore = cookies();
  return cookieStore.get('navCollapsed')?.value === 'true';
}
