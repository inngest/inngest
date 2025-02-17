import type { ReactNode } from 'react';

import { ClientSideProviders } from './ClientSideProviders';

type OrganizationActiveLayoutProps = {
  children: ReactNode;
};

export default async function OrganizationActiveLayout({
  children,
}: OrganizationActiveLayoutProps) {
  return (
    <ClientSideProviders>
      {children}
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
    </ClientSideProviders>
  );
}
