import Container from "./Container";
import SectionHeader from "./SectionHeader";

export default function DevUI() {
  return (
    <div className="overflow-hidden pb-60 -mb-60">
      <div>
        <Container className="mt-60 -mb-30">
          <SectionHeader
            title={
              <span className="lg:flex gap-2 items-end text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
                Tools for "lightspeed development*"{" "}
                <span className="inline-block text-sm text-slate-500 tracking-normal ">
                  *actual words a customer used
                </span>
              </span>
            }
            lede="Our dev server runs on your machine providing you instant feedback
            and debugging tools so you can build serverless functions with
            events like never before possible."
          />
        </Container>
      </div>

      <div className="w-screen max-w-screen relative -mt-10 md:-mt-20 lg:-mt-32 xl:-mt-48">
        <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[500px] h-[200px] md:h-[400px] lg:h-[500px] bg-sky-500/20 absolute rounded-full left-1/2 -translate-x-[20%] translate-y-[40%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[450px] lg:w-[550px] h-[200px] md:h-[450px] lg:h-[550px] bg-indigo-500/30 absolute rounded-full left-1/2 -translate-x-[100%] translate-y-[40%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[300px] lg:w-[400px] h-[200px] md:h-[300px] lg:h-[400px] bg-purple-500/30 absolute rounded-full left-1/2 translate-x-[50%] translate-y-[40%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[500px] h-[200px] md:h-[400px] lg:h-[500px] bg-indigo-500/10 absolute rounded-full bottom-0 left-1/2 -translate-x-[20%] -translate-y-[12%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[400px] lg:w-[550px] h-[200px] md:h-[400px] lg:h-[550px] bg-purple-500/10 absolute rounded-full bottom-0 left-1/2 -translate-x-[100%] translate-y-[6%] "></div>
        <div className=" blur-3xl w-[200px] md:w-[200px] lg:w-[400px] h-[200px] md:h-[200px] lg:h-[400px] bg-blue-500/10 absolute rounded-full bottom-0 left-1/2 translate-x-[50%] translate-y-[6%] "></div>
        <div className="overflow-x-hidden overflow-y-hidden w-screen">
          <img
            src="/assets/DevUI.png"
            className="rounded-sm shadow-none m-auto w-screen relative z-10 scale-110 origin-center max-w-[1723px]"
          />
        </div>
      </div>
    </div>
  );
}
