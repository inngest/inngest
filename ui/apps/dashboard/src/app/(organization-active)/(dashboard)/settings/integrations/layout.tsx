import type { Route } from 'next';
import { Link } from '@inngest/components/Link';

import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';

type IntegrationsLayoutProps = {
  children: React.ReactNode;
};

export default async function IntegrationsLayout({ children }: IntegrationsLayoutProps) {
  const newIntegrations = await getBooleanFlag('new-integrations');

  return newIntegrations ? (
    <>{children}</>
  ) : (
    <div className="flex min-h-0 divide-x divide-slate-100">
      <nav className="w-60 shrink-0 p-8">
        <ul>
          <li>
            <Link
              className="block w-full rounded-md bg-slate-100 px-3 py-2 text-sm font-semibold"
              href={'/settings/integrations/vercel' as Route}
            >
              Vercel
            </Link>
          </li>
        </ul>
      </nav>
      <main className="flex-1 overflow-y-scroll">{children}</main>
    </div>
  );
}
