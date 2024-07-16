import type { ReactNode } from 'react';

import { getEnvs } from '@/components/Environments/data';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import Navigation from '@/components/Navigation/Navigation';
import { URQLProvider } from '@/queries/URQLProvider';
import IncidentBanner from './IncidentBanner';

type OrganizationActiveLayoutProps = {
  children: ReactNode;
};

export default async function OrganizationActiveLayout({
  children,
}: OrganizationActiveLayoutProps) {
  const newIANav = await getBooleanFlag('new-ia-nav');

  return (
    <URQLProvider>
      {true ? (
        <div className="flex w-full flex-row justify-start">
          <div className="bg-canvasBase border-subtle h-screen w-[224px] shrink-0 border-r">
            <Navigation />
          </div>
          <div className="flex w-full flex-col">
            <IncidentBanner />
            {children}
          </div>
        </div>
      ) : (
        <>
          <IncidentBanner />
          {children}
        </>
      )}
      <script
        dangerouslySetInnerHTML={{
          __html: `
            (function (m, a, z, e) {
              var s, t;
              try {
                t = m.sessionStorage.getItem('maze-us');
              } catch (err) {}

              if (!t) {
                t = new Date().getTime();
                try {
                  m.sessionStorage.setItem('maze-us', t);
                } catch (err) {}
              }

              s = a.createElement('script');
              s.src = z + '?apiKey=' + e;
              s.async = true;
              a.getElementsByTagName('head')[0].appendChild(s);
              m.mazeUniversalSnippetApiKey = e;
            })(window, document, 'https://snippet.maze.co/maze-universal-loader.js', 'ca524e7a-1966-427b-9289-c48994e3b9df');
          `,
        }}
      ></script>
    </URQLProvider>
  );
}
