import React, { useEffect } from "react";
import Head from "next/head";
import Script from "next/script";
import { v4 as uuid } from "uuid";
import "../styles/globals.css";

import PageBanner from "../shared/PageBanner";

function MyApp({ Component, pageProps }) {
  return (
    <>
      <Head>
        <link rel="icon" href="/favicon.png" />
      </Head>
      <PageBanner href="/docs/function-ide-guide?ref=page-banner">
        Our new <strong>Function IDE</strong>: Build, test and deploy functions
        right from your browser - Read the quickstart guide &rsaquo;
      </PageBanner>
      <Component {...pageProps} />
      <Script
        id="js-inngest-sdk"
        strategy="afterInteractive"
        src="/inngest-sdk.js"
        onLoad={() => {
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
