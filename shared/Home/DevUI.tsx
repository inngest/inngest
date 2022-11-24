import Container from "./Container";

export default function DevUI() {
  return (
    <>
      <div>
        <Container className="mt-60 -mb-30">
          <h2 className="lg:flex gap-2 items-end text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
            Tools for "lightspeed development*"{" "}
            <span className="inline-block text-sm text-slate-500 tracking-normal ">
              *actual words a customer used
            </span>
          </h2>
          <p className="text-slate-300 font-light max-w-md lg:max-w-xl text-sm lg:text-base leading-5 lg:leading-7">
            Our dev server runs on your machine providing you instant feedback
            and debugging tools so you can build serverless functions with
            events like never before possible.
          </p>
        </Container>
      </div>

      <div className="w-screen overflow-hidden">
        <img
          src="/assets/DevUI.png"
          className="rounded-sm shadow-none m-auto w-screen -mt-40 relative z-10 scale-110 origin-center max-w-[1723px]"
        />
      </div>
    </>
  );
}
