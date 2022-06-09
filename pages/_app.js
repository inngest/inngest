import React, { useEffect } from "react";
import Head from "next/head";
import { useRouter } from "next/router";
import Script from "next/script";
import { v4 as uuid } from "uuid";
import { trackPageView } from "../utils/tracking";
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
      trackPageView(url);
    };
    router.events.on("routeChangeComplete", handleRouteChange);
    return () => {
      router.events.off("routeChangeComplete", handleRouteChange);
    };
  }, [router.events]);
  return (
    <>
      <Head>
        <link rel="icon" href="/favicon-may-2022.png" />
      </Head>
      <PageBanner href="/docs/using-the-inngest-cli?ref=page-banner">
        Introducing the Inngest CLI: build, test, and ship serverless functions
        locally
      </PageBanner>
      <Component {...pageProps} />
      <Script
        id="js-inngest-sdk"
        strategy="afterInteractive"
        src="/inngest-sdk.js"
        onLoad={() => {
          return;
          Inngest.init(process.env.NEXT_PUBLIC_INNGEST_KEY);
          let firstTouch = false;
          const anonId = () => {
            let id = window.localStorage.getItem("inngest-anon-id");
            firstTouch = !id;
            if (!id) {
              id = uuid();
              window.localStorage.setItem("inngest-anon-id", id);
            }
            return id;
          };
          let ref = null;
          try {
            const urlParams = new URLSearchParams(window.location.search);
            ref = urlParams.get("ref");
          } catch (e) {}
          Inngest.identify({ anonymous_id: anonId() });
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
