import Document, { Html, Head, Main, NextScript } from "next/document";

export default class MyDocument extends Document {
  render() {
    return (
      <Html>
        <Head>
          <link rel="icon" href="/favicon-may-2022.png" />
          <link
            rel="stylesheet"
            href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.4.0/styles/github-dark.min.css"
          />
          <script
            defer
            src="https://static.cloudflareinsights.com/beacon.min.js"
            data-cf-beacon='{"token": "e2fa9f28c34844e4a0d29351b8730579"}'
          ></script>
          <script
            // We use a simple array queue to send any events after the SDK is loaded
            // These are sent onLoad where the script is loaded in _app.js
            type="text/javascript"
            dangerouslySetInnerHTML={{
              __html: `
                window._inngestQueue = [];
                if (typeof window.Inngest === "undefined") {
                  window.Inngest = { event: function(p){ window._inngestQueue.push(p); } };
                }
              `,
            }}
          />
        </Head>
        <body className="light-theme">
          <script
            type="text/javascript"
            dangerouslySetInnerHTML={{
              __html: `
               (function(){
                // Any page with a base path matching this will enable light mode
                const THEME_SUPPORT = [
                  "/docs",
                  "/blog"
                ];
                 const hasThemeSupport = !!THEME_SUPPORT.find(function(p) {
                  return document.location.pathname.indexOf(p) === 0;
                 });
                 const theme = window.localStorage.getItem("theme");
                 if (hasThemeSupport && theme) {
                  document.body.classList.add(theme + "-theme");
                 }
               })();
              `,
            }}
          />
          <Main />
          <NextScript />
        </body>
      </Html>
    );
  }
}
