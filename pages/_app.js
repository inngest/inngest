import React, { useEffect } from "react";
import Head from "next/head";
import { useRouter } from "next/router";
import Script from "next/script";
import { v4 as uuid } from "uuid";
import { trackPageView } from "../utils/tracking";
import { useAnonId } from "../shared/trackingHooks";
import "../styles/globals.css";

import PageBanner from "../shared/PageBanner";

function MyApp({ Component, pageProps }) {
  const router = useRouter();
  useEffect(() => {
    if (pageProps.htmlClassName) {
      document.getElementsByTagName("html")[0].className =
        pageProps.htmlClassName;
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

  const { anonId, existing } = useAnonId();

  return (
    <>
      <Head>
        {pageProps?.meta?.title && (
          <>
            <title>Inngest → {pageProps.meta.title}</title>
            <meta
              property="og:title"
              content={`Inngest - ${pageProps.meta.title}`}
            />
          </>
        )}
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
        <meta
          property="og:image"
          content={pageProps?.meta?.image || "/assets/img/og-image-default.jpg"}
        />
        <meta
          property="og:url"
          content={`https://www.inngest.com${router.pathname}`}
        />
      </Head>
      <PageBanner href="/blog/open-source-event-driven-queue?ref=page-banner">
        Announcing our open source plans for the Inngest event-driven queue
      </PageBanner>
      <Component {...pageProps} />
      <Script
        id="js-inngest-sdk"
        strategy="afterInteractive"
        src="/inngest-sdk.js"
        onLoad={() => {
          Inngest.init(process.env.NEXT_PUBLIC_INNGEST_KEY);
          // The hook should tell us if the anon id is an existing one, or it's just been set
          const firstTouch = !existing;
          let ref = null;
          try {
            const urlParams = new URLSearchParams(window.location.search);
            ref = urlParams.get("ref");
          } catch (e) {}
          Inngest.identify({ anonymous_id: anonId });
          // See tracking for next/link based transitions in tracking.ts
          Inngest.event({
            name: "website/page.viewed",
            data: {
              first_touch: firstTouch,
              ref: ref,
            },
          });
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
