import Container from "./Container";

export default function EventDriven() {
  return (
    <>
      <Container className="mt-20 mb-12">
        <h2 className="text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
          Event driven, made simple
        </h2>
        <p className="text-slate-300 font-light max-w-md lg:max-w-xl text-sm lg:text-base leading-5 lg:leading-7">
          Add Inngest to your stack in a few lines for code, then deploy to your
          existing provider. You don't have to change anything to get started.
        </p>
      </Container>

      <div className="bg-gradient-to-r from-slate-1000/0  to-slate-900 pb-32 relative z-10">
        <Container className="px-20">
          <div className="py-16">
            <h3 className="text-lg xl:text-2xl text-slate-50 mb-3">
              Write code, send events
            </h3>
            <p className="text-slate-400 text-sm max-w-lg leading-5 lg:leading-7 lg:text-base">
              Use the Inngest SDK to define functions that are triggered by
              events from your app or anywhere on the internet.
            </p>
            <code className="text-xs mr-5 text-slate-50 mt-8 inline-block bg-slate-800/50 px-4 py-2 rounded-lg">
              <span className="text-slate-500 mr-2">$</span>
              npm install inngest
            </code>
          </div>
          {/* <SendEventsImg /> */}
        </Container>
      </div>

      <Container className="flex flex-col lg:flex-row  gap-6 lg:gap-8 xl:gap-16 -mt-24 ">
        <div className="lg:w-1/2 relative md:mr-40 lg:mr-0">
          <div className="absolute inset-0 rounded-lg bg-blue-500 opacity-20 rotate-2 -z-0 scale-x-[110%] mx-5"></div>
          <div
            style={{
              backgroundImage: "url(/assets/footer/footer-grid.svg)",
              backgroundSize: "cover",
              backgroundPosition: "right -60px top -160px",
              backgroundRepeat: "no-repeat",
            }}
            className=" flex flex-col justify-between text-center bg-blue-500/90 rounded-xl py-6 lg:py-11 px-4 xl:px-16 relative w-full h-full"
          >
            <div>
              <h4 className="text-white text-xl lg:text-2xl font-medium tracking-tight mb-2">
                Use with your favorite frameworks
              </h4>
              <p className="text-sky-200 text-sm ">
                Write your code directly within your existing codebase.
              </p>
            </div>
            <div>logos</div>
          </div>
        </div>

        <div className="lg:w-1/2 relative md:ml-40 lg:ml-0">
          <div className="absolute inset-0 rounded-lg bg-purple-500 opacity-20 rotate-2 -z-0 scale-x-[110%] mx-5"></div>
          <div
            style={{
              backgroundImage: "url(/assets/footer/footer-grid.svg)",
              backgroundSize: "cover",
              backgroundPosition: "right -60px top -160px",
              backgroundRepeat: "no-repeat",
            }}
            className=" flex flex-col justify-between text-center bg-purple-500/90 rounded-xl py-6 lg:py-11 px-4 xl:px-16 relative w-full h-full"
          >
            <div>
              <h4 className="text-white text-xl lg:text-2xl font-medium tracking-tight mb-2">
                Deploy functions anywhere
              </h4>
              <p className="text-purple-100 text-sm ">
                Inngest calls your code, securely, as events are received.
                <br />
                Keep shipping your code as you do today.
              </p>
            </div>
            <div>logos</div>
          </div>
        </div>
      </Container>
    </>
  );
}
