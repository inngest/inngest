'use server';

import { cookies } from 'next/headers';

export async function toggleNav() {
  const cookieStore = cookies();
  const collapsed = cookieStore.get('navCollapsed')?.value === 'true';
  cookieStore.set('navCollapsed', collapsed ? 'false' : 'true');
}
