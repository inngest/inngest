import Head from 'next/head';
import Link from 'next/link';
import { Router } from 'next/router';
import { MDXProvider } from '@mdx-js/react';
import { motion } from 'framer-motion';
import * as mdxComponents from 'src/shared/Docs/mdx';

import { getOpenGraphImageURL } from '../../utils/social';
import Logo from '../Icons/Logo';
import { Footer } from './Footer';
import { Header } from './Header';
import { Home } from './Home';
import { useMobileNavigationStore } from './MobileNavigation';
import { Navigation } from './Navigation';
import { Prose } from './Prose';
import { SectionProvider } from './SectionProvider';

// Unsure if this should be here or in the _app and conditionally run only on docs
function onRouteChange() {
  useMobileNavigationStore.getState().close();
}

Router.events.on('hashChangeStart', onRouteChange);
Router.events.on('routeChangeComplete', onRouteChange);
Router.events.on('routeChangeError', onRouteChange);

export type Props = {
  children: React.ReactNode;
  sections: [];
  /* The title automatically pulled from the h1 tag in each mdx file */
  title: string;
  /* The optional title used for meta tags (set via export const metaTitle = '...') */
  metaTitle?: string;
  /* The optional description */
  description?: string;
};

export function Layout({ children, sections = [], title, metaTitle, description }: Props) {
  const siteTitle = `Inngest Documentation`;
  const preferredTitle: string = metaTitle || title || siteTitle;
  const pageTitle =
    preferredTitle === siteTitle ? preferredTitle : `${preferredTitle} - ${siteTitle}`;
  const metaDescription = description || `Inngest documentation for ${preferredTitle}`;
  const metaImage = getOpenGraphImageURL({ title: preferredTitle });

  return (
    <div className="dark:bg-slate-1000">
      <MDXProvider components={mdxComponents}>
        <Head>
          <title>{pageTitle}</title>
          <meta name="description" content={metaDescription}></meta>
          <meta property="og:title" content={pageTitle} />
          <meta property="og:description" content={metaDescription} />
          <meta property="og:image" content={metaImage} />
          <meta property="og:type" content="article" />
          <meta name="twitter:card" content="summary_large_image" />
          <meta name="twitter:site" content="@inngest" />
          <meta name="twitter:title" content={pageTitle} />
          <meta name="twitter:image" content={metaImage} />

          <script dangerouslySetInnerHTML={{ __html: modeScript }} />
        </Head>
        <SectionProvider sections={sections}>
          <div className="lg:ml-72 xl:ml-80">
            <motion.header
              layoutScroll
              className="fixed inset-y-0 left-0 z-40 contents w-72 overflow-y-auto border-r border-slate-900/10 px-6 pb-8 pt-4 lg:block xl:w-80 dark:border-white/10"
            >
              <div className="hidden lg:flex">
                <a href="/" className="group/logo flex items-center gap-1.5">
                  <Logo className="w-20 text-indigo-500 dark:text-white" />
                  <span className="transition-color mb-0.5 text-base font-semibold text-slate-700 group-hover/logo:text-slate-500 dark:text-indigo-400 dark:group-hover/logo:text-white">
                    Docs
                  </span>
                </a>
              </div>
              <Header />
              <Navigation className="hidden lg:mt-6 lg:block" />
            </motion.header>
            <div className="relative px-4 pt-14 sm:px-6 lg:px-8">
              <main className="py-16">
                <Prose as="article">{children}</Prose>
              </main>
              <Footer />
            </div>
          </div>
        </SectionProvider>
      </MDXProvider>
    </div>
  );
}

const modeScript = `
  // change to "let = darkModeMediaQuery" if/when this moves to the _document
  window.darkModeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

  updateMode()
  window.darkModeMediaQuery.addEventListener('change', updateModeWithoutTransitions)
  window.addEventListener('storage', updateModeWithoutTransitions)

  function updateMode() {
    let isSystemDarkMode = window.darkModeMediaQuery.matches
    let isDarkMode = window.localStorage.isDarkMode === 'true' || (!('isDarkMode' in window.localStorage) && isSystemDarkMode)

    if (isDarkMode) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }

    if (isDarkMode === isSystemDarkMode) {
      delete window.localStorage.isDarkMode
    }
  }

  function disableTransitionsTemporarily() {
    document.documentElement.classList.add('[&_*]:!transition-none')
    window.setTimeout(() => {
      document.documentElement.classList.remove('[&_*]:!transition-none')
    }, 0)
  }

  function updateModeWithoutTransitions() {
    disableTransitionsTemporarily()
    updateMode()
  }
`;
