import Link from "next/link";
import Head from "next/head"
import { MDXProvider } from "@mdx-js/react";
import { motion } from "framer-motion";

import * as mdxComponents from "src/shared/NewDocs/mdx";
import { Footer } from "./Footer";
import { Header } from "./Header";
import { Logo } from "./Logo";
import { Navigation } from "./Navigation";
import { Prose } from "./Prose";
import { SectionProvider } from "./SectionProvider";

export function Layout({ children, sections = [] }) {
  return (
    <MDXProvider components={mdxComponents}>

      <Head>
        <script dangerouslySetInnerHTML={{ __html: modeScript }} />
      </Head>
      <SectionProvider sections={sections}>
        <div className="lg:ml-72 xl:ml-80">
          <motion.header
            layoutScroll
            className="fixed inset-y-0 left-0 z-40 contents w-72 overflow-y-auto border-r border-slate-900/10 px-6 pt-4 pb-8 dark:border-white/10 lg:block xl:w-80"
          >
            <div className="hidden lg:flex">
              <Link href="/" aria-label="Home">
                <Logo className="h-6" />
              </Link>
            </div>
            <Header />
            <Navigation className="hidden lg:mt-10 lg:block" />
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
`
