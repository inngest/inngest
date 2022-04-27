import Document, { Html, Head, Main, NextScript } from "next/document";

export default class MyDocument extends Document {
  render() {
    const pageProps = this.props?.__NEXT_DATA__?.props?.pageProps;
    const { htmlClassName } = pageProps;
    return (
      <Html className={htmlClassName || ""}>
        <Head />
        <body>
          <script
            type="text/javascript"
            dangerouslySetInnerHTML={{
              __html: `
               (function(){
                // Any page with a base path matching this will enable light mode
                const LIGHT_THEME_SUPPORT = [
                  "/docs",
                  "/blog"
                ];
                 const hasLightTheme = !!LIGHT_THEME_SUPPORT.find(function(p) {
                  return document.location.pathname.indexOf(p) === 0;
                 });
                 const theme = window.localStorage.getItem("theme");
                 if (hasLightTheme && theme) {
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
