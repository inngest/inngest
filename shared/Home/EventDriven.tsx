import Container from "./Container";
import SendEvents from "./HomeImg/SendEvents";
import SectionHeader from "./SectionHeader";
import NextJs from "../Icons/NextJs";
import CloudflarePages from "../Icons/CloudflarePages";
import RedwoodJs from "../Icons/RedwoodJs";
import Express from "../Icons/Express";
import Netlify from "../Icons/Netlify";
import Vercel from "../Icons/Vercel";
import Logo from "../Icons/Logo";
import Cloudflare from "../Icons/Cloudflare";

export default function EventDriven() {
  return (
    <>
      <Container className="mt-20 mb-12">
        <SectionHeader
          title="Event driven, made simple"
          lede="Add Inngest to your stack in a few lines for code, then deploy to your
          existing provider. You don't have to change anything to get started."
        />
      </Container>

      <div className="bg-gradient-to-r from-slate-1000/0  to-slate-900 pb-32 relative z-10">
        <Container className="px-20 h-[440px]">
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
          <SendEvents />
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
            <div className="flex items-center w-full justify-between mt-6 px-3 flex-wrap ">
              <NextJs />
              <Express />
              <RedwoodJs />
              <CloudflarePages />
            </div>
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
            className=" flex flex-col justify-between text-center bg-purple-500/90 rounded-xl relative w-full h-full"
          >
            <div className="pt-8 lt:p-11 px-4 xl:px-16 ">
              <h4 className="text-white text-xl lg:text-2xl font-medium tracking-tight mb-2">
                Deploy functions anywhere
              </h4>
              <p className="text-purple-100 text-sm ">
                Inngest calls your code, securely, as events are received.
                <br />
                Keep shipping your code as you do today.
              </p>
            </div>
            <div className="flex items-center mt-6 mb-8 px-8 lg:gap-4 lg:flex-wrap m-auto">
              <div className="">
                <Netlify />
              </div>
              <div className="">
                <Vercel />
              </div>
              <div className="">
                <Logo className="text-slate-200" />
              </div>
              <div className=" relative -top-1">
                <Cloudflare />
              </div>
            </div>
          </div>
        </div>
      </Container>
    </>
  );
}
