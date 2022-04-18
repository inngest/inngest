import Head from "next/head";
import Script from "next/script";
import { v4 as uuid } from "uuid";
import "../styles/globals.css";

function MyApp({ Component, pageProps }) {
  return (
    <>
      <Head>
        <link rel="icon" href="/favicon.png" />
      </Head>
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
          Inngest.identify({ anonymous_id: anonId() });
          Inngest.event({
            name: "website/page.viewed",
            data: {
              first_touch: firstTouch,
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
