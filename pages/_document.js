import Document, { Html, Head, Main, NextScript } from "next/document";

export default class MyDocument extends Document {
  render(props) {
    const pageProps = props?.__NEXT_DATA__?.props?.pageProps;
    const isDarkMode =
      typeof pageProps?.isDarkMode !== "undefined"
        ? pageProps.isDarkMode
        : true;
    return (
      <Html className={pageProps?.htmlClassName || ""}>
        <Head />
        <body className={isDarkMode ? "dark-mode" : "light-mode"}>
          <Main />
          <NextScript />
        </body>
      </Html>
    );
  }
}
