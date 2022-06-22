import Document, { Html, Head, Main, NextScript } from "next/document";

export default class MyDocument extends Document {
  render() {
    // const pageProps = this.props?.__NEXT_DATA__?.props?.pageProps;
    // const { htmlClassName } = pageProps;
    return (
      <Html className={"OK"}>
        <Head>
          <link
            rel="stylesheet"
            href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.4.0/styles/github-dark.min.css"
          />
        </Head>
        <body className="light-theme XYZ">
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
