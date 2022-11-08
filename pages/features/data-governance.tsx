import React from "react";
import Nav from "../../shared/nav";
import Tag from "../../shared/Tag";
import Button from "src/shared/Button";
import Footer from "src/shared/Footer";

export default function () {
  const [hoverVal, setHoverVal] = React.useState("");

  const hover = (e: React.SyntheticEvent<HTMLDivElement>) => {
    const str = (e.target as HTMLDivElement).getAttribute("data-hover");
    if (typeof str === "string") {
      // Only set hover value if we're targeting the div element.
      // This is emptied via onMouseLeave within the parent container.
      setHoverVal(str);
    }
  };

  return (
    <div>
      <Nav sticky={true} />

      <div className="container mx-auto py-24">
        <div className="max-w-2xl px-4 lg:px-0 mx-auto text-center">
          <Tag className="mb-4">Data governance</Tag>
          <h1>Guarantee good data, everywhere.</h1>
          <p className="py-4 font-lg">
            A single platform that allows you to inspect, validate, fix, and
            enrich data as it’s processed — ensuring your data is correct in
            every platform, everywhere it's used.
          </p>

          <Button
            kind="primary"
            href="/sign-up"
            size="medium"
            style={{ display: "inline-block" }}
            className="mt-4"
          >
            Get started for free
          </Button>
        </div>
      </div>

      <div className="mx-auto center-dash-purple">
        <div className="container mx-auto text-center">
          <Tag>Feature highlights</Tag>
        </div>
      </div>
      <div
        className="container mx-auto max-w-4xl grid lg:grid-cols-5 pt-16 text-center lg:px-0 sm:px-6 feature-highlights"
        onMouseOver={hover}
        onMouseLeave={() => setHoverVal("")}
      >
        <div
          className="flex flex-col items-center hover:shadow-xl hover:bg-white rounded-sm p-4"
          data-hover="Write custom transforms to adapt data as it's processed"
        >
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M15.7071 6.29289C15.3166 5.90237 14.6834 5.90237 14.2929 6.29289C13.9024 6.68342 13.9024 7.31658 14.2929 7.70711L15.7071 6.29289ZM20 12L20.7071 12.7071C21.0976 12.3166 21.0976 11.6834 20.7071 11.2929L20 12ZM14.2929 16.2929C13.9024 16.6834 13.9024 17.3166 14.2929 17.7071C14.6834 18.0976 15.3166 18.0976 15.7071 17.7071L14.2929 16.2929ZM8.29289 17.7071C8.68342 18.0976 9.31658 18.0976 9.70711 17.7071C10.0976 17.3166 10.0976 16.6834 9.70711 16.2929L8.29289 17.7071ZM4 12L3.29289 11.2929C2.90237 11.6834 2.90237 12.3166 3.29289 12.7071L4 12ZM9.70711 7.70711C10.0976 7.31658 10.0976 6.68342 9.70711 6.29289C9.31658 5.90237 8.68342 5.90237 8.29289 6.29289L9.70711 7.70711ZM14.2929 7.70711L19.2929 12.7071L20.7071 11.2929L15.7071 6.29289L14.2929 7.70711ZM19.2929 11.2929L14.2929 16.2929L15.7071 17.7071L20.7071 12.7071L19.2929 11.2929ZM9.70711 16.2929L4.70711 11.2929L3.29289 12.7071L8.29289 17.7071L9.70711 16.2929ZM4.70711 12.7071L9.70711 7.70711L8.29289 6.29289L3.29289 11.2929L4.70711 12.7071Z"
              fill="#070707"
            />
          </svg>

          <p className="pt-4">Inline JavaScript (ES6+) transforms</p>
        </div>
        <div
          className="flex flex-col items-center hover:shadow-xl hover:bg-white rounded-sm p-4"
          data-hover="Version each event with its own schema to control your data as it changes over time"
        >
          <svg
            width="24"
            height="24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="m21 8 .555.832a1 1 0 0 0 0-1.664L21 8Zm-9 6-.555.832a1 1 0 0 0 1.11 0L12 14ZM3 8l-.555-.832a1 1 0 0 0 0 1.664L3 8Zm9-6 .555-.832a1 1 0 0 0-1.11 0L12 2Zm9.555 10.832a1 1 0 0 0-1.11-1.664l1.11 1.664ZM12 18l-.555.832a1 1 0 0 0 1.11 0L12 18Zm-8.445-6.832a1 1 0 0 0-1.11 1.664l1.11-1.664Zm18 5.664a1 1 0 0 0-1.11-1.664l1.11 1.664ZM12 22l-.555.832a1 1 0 0 0 1.11 0L12 22Zm-8.445-6.832a1 1 0 0 0-1.11 1.664l1.11-1.664Zm16.89-8-9 6 1.11 1.664 9-6-1.11-1.664Zm-7.89 6-9-6-1.11 1.664 9 6 1.11-1.664Zm-9-4.336 9-6-1.11-1.664-9 6 1.11 1.664Zm7.89-6 9 6 1.11-1.664-9-6-1.11 1.664Zm9 8.336-9 6 1.11 1.664 9-6-1.11-1.664Zm-7.89 6-9-6-1.11 1.664 9 6 1.11-1.664Zm7.89-2-9 6 1.11 1.664 9-6-1.11-1.664Zm-7.89 6-9-6-1.11 1.664 9 6 1.11-1.664Z"
              fill="#070707"
            />
          </svg>

          <p className="pt-4">Event versioning and&nbsp;management</p>
        </div>
        <div
          className="flex flex-col items-center hover:shadow-xl hover:bg-white rounded-sm p-4"
          data-hover="Enforce schemas for each event, and store invalid events in quarantine for inspection and post-processing"
        >
          <svg
            width="24"
            height="24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M15.4 21v-1 1ZM4.6 21v1-1ZM17 19.4h1-1Zm-.11 1.054.892.454-.891-.454Zm-.436.437.454.891-.454-.891Zm0-9.782L16 12l.454-.891ZM17 12.6h-1 1Zm-.11-1.054L16 12l.89-.454ZM3.11 20.454 4 20l-.891.454Zm.437.437L4 20l-.454.891ZM17 14a1 1 0 1 0 0 2v-2ZM6 9a1 1 0 0 0 2 0H6Zm15-2.4h-1 1Zm-.11-1.054.892-.454-.891.454Zm-.436-.437.454-.891-.454.891Zm0 9.782.454.891-.454-.891ZM21 13.4h1-1Zm-.11 1.054L20 14l.89.454Zm-11.68-2.84a1 1 0 1 0 1.58-1.228l-1.58 1.228Zm-.285-1.996-.79.614.79-.614Zm-5.38-.509L4 10l-.454-.891Zm-.436.437L4 10l-.891-.454Zm5.44-.35.595-.804-.594.804Zm-.324-.159.27-.963-.27.963Zm4.986-3.423a1 1 0 1 0 1.578-1.228l-1.578 1.228Zm-.286-1.996.79-.614-.79.614Zm-5.38-.509L8 4l-.454-.891Zm-.436.437L8 4l-.891-.454Zm5.44-.35.596-.804-.595.804Zm-.324-.159.27-.963-.27.963ZM4 19.4V11H2v8.4h2ZM3 12h12.4v-2H3v2Zm12.4 8H4.6v2h10.8v-2Zm.6-.6c0 .297-.001.459-.01.575-.01.105-.02.082.01.025l1.782.908c.138-.271.182-.541.2-.77.018-.217.018-.475.018-.738h-2Zm-.6 2.6c.263 0 .521 0 .738-.017.229-.019.499-.063.77-.201L16 20c.058-.03.08-.019-.025-.01-.116.01-.279.01-.575.01v2Zm.6-2 .908 1.782c.378-.192.683-.5.874-.874L16 20Zm-.6-8c.296 0 .459 0 .576.01.105.009.082.02.024-.01l.908-1.782a2.023 2.023 0 0 0-.77-.201c-.217-.018-.475-.017-.738-.017v2Zm2.6.6c0-.264 0-.521-.017-.738a2.023 2.023 0 0 0-.201-.77L16 12c-.03-.057-.02-.08-.01.025.009.116.01.278.01.575h2Zm-2-.6 1.782-.908a1.998 1.998 0 0 0-.874-.874L16 12ZM2 19.4c0 .264 0 .521.017.738.019.229.063.499.201.77L4 20c.03.058.019.08.01-.025A8.194 8.194 0 0 1 4 19.4H2Zm2.6.6c-.297 0-.46 0-.576-.01-.104-.009-.082-.02-.024.01l-.908 1.782c.271.138.54.182.77.201.216.018.474.017.738.017v-2Zm-2.382.908a2 2 0 0 0 .874.874L4 20l-1.782.908ZM16 12.6v6.8h2v-6.8h-2Zm1 3.4h2.4v-2H17v2Zm2.4-12H7v2h12.4V4ZM6 5v4h2V5H6Zm16 1.6c0-.264 0-.521-.017-.738a2.022 2.022 0 0 0-.201-.77L20 6c-.03-.058-.02-.08-.01.025.009.116.01.278.01.575h2ZM19.4 6c.297 0 .459 0 .576.01.105.009.082.02.024-.01l.908-1.782a2.023 2.023 0 0 0-.77-.201C19.921 3.999 19.663 4 19.4 4v2Zm2.382-.908a1.999 1.999 0 0 0-.874-.874L20 6l1.782-.908ZM19.4 16c.264 0 .521 0 .738-.017.229-.019.499-.063.77-.201L20 14c.058-.03.08-.019-.024-.01-.117.01-.28.01-.576.01v2Zm.6-2.6c0 .297-.001.459-.01.575-.01.105-.02.082.01.025l1.782.908c.138-.271.182-.541.2-.77.018-.217.018-.474.018-.738h-2Zm.908 2.382c.378-.193.683-.5.874-.874L20 14l.908 1.782ZM22 13.4V6.6h-2v6.8h2Zm-11.21-3.014L9.713 9.004l-1.579 1.228 1.076 1.382 1.578-1.228ZM7.661 8H4.6v2h3.062V8ZM2 10.6v.4h2v-.4H2ZM4.6 8c-.264 0-.522 0-.739.017a2.021 2.021 0 0 0-.77.201L4 10c-.058.03-.08.019.024.01.117-.01.28-.01.576-.01V8ZM4 10.6c0-.297 0-.459.01-.575.009-.105.02-.082-.01-.025l-1.782-.908a2.021 2.021 0 0 0-.201.77c-.018.217-.017.474-.017.738h2Zm-.908-2.382a2 2 0 0 0-.874.874L4 10l-.908-1.782Zm6.622.786c-.142-.183-.323-.43-.57-.612L7.955 10c-.024-.018-.028-.028.002.007.036.043.084.104.178.225l1.58-1.228ZM7.662 10c.153 0 .23 0 .287.003.046.003.036.005.007-.003l.539-1.926C8.199 7.991 7.892 8 7.662 8v2Zm1.482-1.608a1.998 1.998 0 0 0-.65-.318L7.957 10l1.188-1.608Zm5.645-4.006-1.075-1.382-1.579 1.228 1.076 1.382 1.578-1.228ZM11.662 2H8.6v2h3.062V2ZM6 4.6V5h2v-.4H6ZM8.6 2c-.264 0-.522 0-.739.017a2.021 2.021 0 0 0-.77.201L8 4c-.058.03-.08.019.024.01C8.141 4 8.304 4 8.6 4V2ZM8 4.6c0-.297 0-.459.01-.575C8.02 3.92 8.03 3.943 8 4l-1.782-.908a2.021 2.021 0 0 0-.201.77C5.999 4.079 6 4.336 6 4.6h2Zm-.908-2.382a2 2 0 0 0-.874.874L8 4l-.908-1.782Zm6.622.786c-.142-.183-.323-.43-.57-.612L11.956 4c-.024-.018-.028-.028.002.007.036.043.084.104.178.225l1.58-1.228ZM11.661 4c.154 0 .232 0 .288.003.046.003.036.005.007-.003l.539-1.926c-.296-.083-.603-.074-.833-.074v2Zm1.484-1.608a2 2 0 0 0-.65-.318L11.955 4l1.19-1.608Z"
              fill="#070707"
            />
          </svg>
          <p className="pt-4">Event schemas with data quarantine</p>
        </div>
        <div
          className="flex flex-col items-center hover:shadow-xl hover:bg-white rounded-sm p-4"
          data-hover="Enrich data via external APIs using any programming language"
        >
          <svg
            width="24"
            height="24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M18 17h1-1Zm0-10h-1 1ZM6 17h1-1ZM6 7H5h1Zm13 5a1 1 0 1 0-2 0h2ZM7 12a1 1 0 1 0-2 0h2Zm12 5V7h-2v10h2ZM7 17V7H5v10h2Zm10 0c0 .246-.225.737-1.205 1.227-.92.46-2.26.773-3.795.773v2c1.778 0 3.438-.358 4.69-.984C17.882 19.42 19 18.41 19 17h-2Zm-5 2c-1.535 0-2.876-.313-3.795-.773C7.225 17.737 7 17.246 7 17H5c0 1.411 1.118 2.42 2.31 3.016 1.252.626 2.912.984 4.69.984v-2Zm5-7c0 .246-.225.737-1.205 1.227-.92.46-2.26.773-3.795.773v2c1.778 0 3.438-.358 4.69-.984C17.882 14.42 19 13.41 19 12h-2Zm-5 2c-1.535 0-2.876-.313-3.795-.773C7.225 12.737 7 12.246 7 12H5c0 1.411 1.118 2.42 2.31 3.016 1.252.626 2.912.984 4.69.984v-2Zm0-5c-1.535 0-2.876-.313-3.795-.773C7.225 7.737 7 7.246 7 7H5c0 1.411 1.118 2.42 2.31 3.016 1.252.626 2.912.984 4.69.984V9ZM7 7c0-.246.225-.737 1.205-1.227C9.125 5.313 10.465 5 12 5V3c-1.778 0-3.438.358-4.69.984C6.118 4.58 5 5.59 5 7h2Zm5-2c1.535 0 2.876.313 3.795.773C16.775 6.263 17 6.754 17 7h2c0-1.411-1.118-2.42-2.31-3.016C15.438 3.358 13.778 3 12 3v2Zm5 2c0 .246-.225.737-1.205 1.227-.92.46-2.26.773-3.795.773v2c1.778 0 3.438-.358 4.69-.984C17.882 9.42 19 8.41 19 7h-2Z"
              fill="#070707"
            />
          </svg>
          <p className="pt-4">Enrichment via external&nbsp;APIs</p>
        </div>
        <div
          className="flex flex-col items-center hover:shadow-xl hover:bg-white rounded-sm p-4"
          data-hover="Proactively alert when events become anomalous or have invalid data, without any setup"
        >
          <svg
            width="24"
            height="24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="m19.196 14.196.707-.707-.707.707Zm.39.39-.707.707.707-.707Zm.362 1.734.947.32-.947-.32Zm-.628.628.32.947-.32-.947Zm.492-2.117.813-.583-.813.583Zm.175.421-.987.162.987-.162ZM12 3V2v1ZM4.013 15.252l.987.162-.987-.162Zm.175-.421-.813-.583.813.583Zm-.136 1.489L5 16l-.948.32Zm.628.628L5 16l-.32.948ZM15 17h1a1 1 0 0 0-1-1v1Zm-6 0v-1a1 1 0 0 0-1 1h1Zm9.62-15.785a1 1 0 0 0-1.203 1.597l1.203-1.597Zm1.635 5.2a1 1 0 1 0 1.835-.798l-1.835.797ZM6.584 2.811A1 1 0 1 0 5.38 1.215l1.204 1.597ZM1.91 5.617a1 1 0 1 0 1.834.797l-1.834-.797Zm3.21 9.676.39-.39-1.415-1.414-.389.39 1.414 1.414ZM6 13.723V10H4v3.722h2ZM18 10v3.722h2V10h-2Zm.49 4.903.389.39 1.414-1.414-.39-.39-1.414 1.414ZM18.585 16H5.414v2h13.172v-2Zm.414-.414c0 .204 0 .316-.005.396-.004.075-.01.061.005.018l1.895.64c.118-.35.105-.75.105-1.054h-2ZM18.586 18c.305 0 .703.013 1.053-.105L19 16c.043-.015.057-.01-.018-.005a8.079 8.079 0 0 1-.396.005v2ZM19 16l.64 1.895a2 2 0 0 0 1.255-1.256L19 16Zm-.121-.707c.064.064.096.096.118.12.019.02.014.016.003.001l1.625-1.166c-.107-.148-.236-.274-.332-.37l-1.414 1.415Zm2.121.293c0-.138.003-.316-.026-.496L19 15.414c-.003-.017-.002-.022-.001.005l.001.167h2Zm-2-.172 1.974-.324a2 2 0 0 0-.35-.842L19 15.414Zm-1-1.692c0 .443.176.868.49 1.181l1.413-1.414a.33.33 0 0 1 .097.233h-2ZM12 4a6 6 0 0 1 6 6h2a8 8 0 0 0-8-8v2Zm-6 6a6 6 0 0 1 6-6V2a8 8 0 0 0-8 8h2Zm-.49 4.903a1.67 1.67 0 0 0 .49-1.18H4a.33.33 0 0 1 .096-.234l1.415 1.414Zm-.51.683.001-.168c.001-.027.002-.021-.001-.004l-1.974-.324c-.03.18-.026.36-.026.496h2Zm-1.293-1.707c-.096.096-.226.22-.332.369L5 15.414c-.01.015-.015.018.003-.001.022-.024.054-.056.118-.12l-1.414-1.414ZM5 15.414l-1.625-1.166a2 2 0 0 0-.349.842L5 15.414Zm-2 .172c0 .305-.013.703.105 1.053L5 16c.015.043.01.056.005-.018A8.099 8.099 0 0 1 5 15.586H3ZM5.414 16c-.204 0-.316 0-.397-.005-.074-.004-.06-.01-.017.005l-.64 1.895c.35.118.75.105 1.054.105v-2Zm-2.31.64a2 2 0 0 0 1.257 1.255L5 16l-1.895.64ZM14 17v1h2v-1h-2Zm-4 1v-1H8v1h2Zm-1 0h6v-2H9v2Zm3 2a2 2 0 0 1-2-2H8a4 4 0 0 0 4 4v-2Zm2-2a2 2 0 0 1-2 2v2a4 4 0 0 0 4-4h-2Zm3.417-15.188a9 9 0 0 1 2.838 3.602l1.835-.797a11 11 0 0 0-3.47-4.402l-1.203 1.597ZM5.38 1.215a11 11 0 0 0-3.47 4.402l1.835.797a9 9 0 0 1 2.839-3.602L5.38 1.215Z"
              fill="#0A0A12"
            />
          </svg>
          <p className="pt-4">Alerting and error reporting&nbsp;built-in</p>
        </div>
      </div>
      <p className="text-center pt-6 text-xs text-slate-500">
        {hoverVal || <>&nbsp;</>}
      </p>

      <div className="container mx-auto grid lg:grid-cols-2 gap-16 px-8 lg:px-0 pt-20 md:pt-44">
        <div>
          <h3 className="pb-4">Develop without errors</h3>
          <p style={{ margin: 0 }} className="pt-3">
            <strong>
              We automatically type-check and validate events sent to Inngest
            </strong>
            , ensuring your functionality is always called correctly. You can be
            sure that your functions are always called with the correct
            arguments, and that data for the rest of your team is correct.
          </p>
          <p style={{ margin: 0 }} className="pt-3">
            And if your data is wrong we'll place your events in quarantine,
            allowing you to fix them and re-process your business logic &mdash;
            for the easiest and safest recover ever.
          </p>
        </div>

        <div className="flex align-center justify-center">
          <img
            src="/assets/overview-simplified.svg"
            alt="Inngest overview"
            height="auto"
            className="self-center"
          />
        </div>
      </div>

      <div className="container mx-auto mt-20 md:mt-48 py-16 text-center background-grid-texture">
        <h3 className="pb-2">Examples, patterns, and guides</h3>
        <p>
          Explore a suite of fully-built examples, plus patterns and guides for
          building rich, reliable functionality
        </p>

        <div className="flex justify-center pt-12">
          <Button kind="outline" href="/quick-starts">
            View examples
          </Button>
          <Button kind="outline" href="/docs">
            View docs
          </Button>
        </div>
      </div>

      <div className="container mx-auto">
        <div className="p-12 md:p-20 mt-12 md:mt-24 rounded-xl bg-slate-100">
          <h2 className="mb-2">Develop faster and safer</h2>
          <p>
            Use our platform to rapidly build functionality driven by events,
            with zero infrastructure and full end-to-end safety.
          </p>
          <div className="grid lg:grid-cols-3 gap-16 mt-20">
            <div>
              <h3 className="pb-4">Powerful</h3>
              <p>
                Transform and enrich data on-the-fly, ensuring your data is as
                clean and powerful as possible. See an entire overview of what
                each event does in your system — no documenting required.
              </p>
            </div>
            <div>
              <h3 className="pb-4">Flexible</h3>
              <p>
                Create unlimited event versions with a full changelog, ensuring
                your event registry can adapt to your business needs as you
                grow.
              </p>
            </div>
            <div>
              <h3 className="pb-4">Reliable</h3>
              <p>
                Store invalid data that doesn’t match your version schema in
                quarantine for debugging and fixing, ensuring <i>all</i> data is
                clean.
              </p>
            </div>
          </div>
        </div>
      </div>

      <Footer />
    </div>
  );
}
