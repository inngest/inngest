import Container from "./Container";
import SectionHeader from "./SectionHeader";

export default function DevUI() {
  return (
    <>
      <div>
        <Container className="mt-60 -mb-30">
          <SectionHeader
            title={
              <h2 className="lg:flex gap-2 items-end text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
                Tools for "lightspeed development*"{" "}
                <span className="inline-block text-sm text-slate-500 tracking-normal ">
                  *actual words a customer used
                </span>
              </h2>
            }
            lede="Our dev server runs on your machine providing you instant feedback
            and debugging tools so you can build serverless functions with
            events like never before possible."
          />
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
