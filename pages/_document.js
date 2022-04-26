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
                 const hasLightMode = document.location.pathname.indexOf("/docs") === 0;
                 const mode = window.localStorage.getItem("screen-mode");
                 if (hasLightMode && mode) {
                  document.body.classList.add(mode + "-mode");
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
