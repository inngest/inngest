import React, { useEffect } from "react";
import Head from "next/head";
import { useRouter } from "next/router";
import Script from "next/script";

import { trackPageView } from "../utils/tracking";
import { getOpenGraphImageURL } from "../utils/social";
import { useAnonId } from "../shared/legacy/trackingHooks";
import "../styles/globals.css";
import * as fullstory from "@fullstory/browser";

import { Layout as DocsLayout } from "../shared/Docs/Layout";

import PageBanner from "../shared/legacy/PageBanner";

function DefaultLayout({ children }) {
  const router = useRouter();
  return (
    <>
      {router.pathname !== "/sign-up" && (
        <PageBanner href="/blog/vercel-integration?ref=page-banner">
          Announcing our new Vercel integration
        </PageBanner>
      )}
      <div className="bg-slate-1000 gradient h-screen w-full absolute top-0 right-0 -z-10 "></div>
      {children}
    </>
  );
}

function MyApp({ Component, pageProps }) {
  const router = useRouter();
  const { anonId, existing } = useAnonId();

  // Temp Layout swapping before we move to "app" dir
  const isDocs = !!router.asPath.match(/^\/docs/);
  const Layout = isDocs ? DocsLayout : DefaultLayout;

  useEffect(() => {
    fullstory.init({ orgId: "o-1CVB8R-na1" });

    const htmlEl = document.getElementsByTagName("html")[0];
    if (pageProps.htmlClassName) {
      htmlEl.className = pageProps.htmlClassName;
    }
    if (pageProps.designVersion) {
      htmlEl.classList.add(`v${pageProps.designVersion}`);
    } else if (isDocs) {
      htmlEl.classList.add(`v2`);
    } else {
      htmlEl.classList.add(`v1`);
    }
    if (isDocs) {
      htmlEl.classList.add(`docs`);
    }
  });
  useEffect(() => {
    const handleRouteChange = (url) => {
      // Track page views when using Next's Link component as it doesn't do a full refresh
      trackPageView(url);
    };
    router.events.on("routeChangeComplete", handleRouteChange);
    return () => {
      router.events.off("routeChangeComplete", handleRouteChange);
    };
  }, [router.events]);

  const title =
    pageProps?.meta?.title ||
    "Reliable serverless background functions on any platform";
  const metaTitle = `Inngest - ${title}`;
  const disableMetadata =
    pageProps?.meta?.disabled === true || router.asPath.match(/^\/docs/);
  // Warn during local dev
  if (
    !disableMetadata &&
    !pageProps?.meta?.title &&
    process.env.NODE_ENV !== "production"
  ) {
    const INNGEST_SDK_URLS = [
      "/api/inngest",
      "/x/inngest",
      "/.redwood/functions/inngest",
      "/.netlify/functions/inngest",
    ];
    // Ignore the dev server polling for functions
    if (!INNGEST_SDK_URLS.includes(router.asPath)) {
      console.warn(
        `WARNING: meta tags are not set for this page, please set via getStaticProps (${router.asPath})`
      );
    }
  }

  const canonicalUrl = `https://www.inngest.com${
    router.asPath === "/" ? "" : router.asPath
  }`.split("?")[0];
  const ogImage =
    pageProps?.meta?.image || getOpenGraphImageURL({ title: title });

  return (
    <>
      <Head>
        {/* Set this for all pages */}
        <meta property="og:url" content={canonicalUrl} />

        {/* Sections of the site like the blog and docs set these using different data */}
        {!disableMetadata && (
          <>
            <title>{metaTitle}</title>
            {pageProps?.meta?.description && (
              <>
                <meta
                  name="description"
                  content={pageProps.meta.description}
                ></meta>
                <meta
                  property="og:description"
                  content={pageProps.meta.description}
                />
              </>
            )}
            <meta property="og:image" content={ogImage} />
            <meta property="og:title" content={metaTitle} />
            <meta name="twitter:card" content="summary_large_image" />
            <meta name="twitter:site" content="@inngest" />
            <meta name="twitter:title" content={metaTitle} />
            {pageProps?.meta?.description && (
              <meta
                name="twitter:description"
                content={pageProps?.meta?.description}
              />
            )}
            <meta name="twitter:image" content={ogImage} />
          </>
        )}
        <link rel="canonical" href={canonicalUrl} />
      </Head>

      <Layout {...pageProps}>
        <Component {...pageProps} />
      </Layout>

      <Script
        id="js-inngest-sdk-script"
        strategy="afterInteractive"
        src="/inngest-sdk.js"
        onLoad={() => {
          Inngest.init(process.env.NEXT_PUBLIC_INNGEST_KEY);
          Inngest.identify({ anonymous_id: anonId });
          // The hook should tell us if the anon id is an existing one, or it's just been set
          const firstTouch = !existing;
          let ref = null;
          try {
            const urlParams = new URLSearchParams(window.location.search);
            ref = urlParams.get("ref");
          } catch (e) {}
          // See tracking for next/link based transitions in tracking.ts
          Inngest.event({
            name: "website/page.viewed",
            data: {
              first_touch: firstTouch,
              ref: ref,
            },
            v: "2022-12-27.1",
          });
          if (typeof window !== "undefined" && window._inngestQueue.length) {
            window._inngestQueue.forEach((p) => {
              // Prevent the double tracking of page views b/c routeChangeComplete
              // is unpredictable.
              if (p.name === "website/page.viewed") return;
              Inngest.event(p);
            });
          }
        }}
      />
      <script
        async
        src={`https://www.googletagmanager.com/gtag/js?id=${process.env.NEXT_PUBLIC_GTAG_ID}`}
      ></script>
      <script
        dangerouslySetInnerHTML={{
          __html: `
        window.dataLayer = window.dataLayer || [];
        function gtag(){dataLayer.push(arguments);}
        gtag('js', new Date());
        gtag('config', '${process.env.NEXT_PUBLIC_GTAG_ID}');
      `,
        }}
      />
    </>
  );
}

export default MyApp;
