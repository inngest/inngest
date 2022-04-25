import Document, { Html, Head, Main, NextScript } from "next/document";

export default class MyDocument extends Document {
  render(props) {
    return (
      <Html>
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
                  console.log("Set mode to", mode);
                  document.body.className = mode + "-mode";
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
