import Link from "next/link";
import HeroImg from "./HomeImg/HeroImg";

export default function Hero() {
  return (
    <div className="relative">
      <div
        style={{
          background: "radial-gradient(circle at center, #13123B, #08090d)",
        }}
        className="absolute w-[200vw]  -translate-x-1/2 -translate-y-1/2 h-[200vw] rounded-full blur-lg opacity-90"
      ></div>

      <div className="max-w-container-desktop mx-auto py-96 xl:py-40 max-h-[600px] xl:max-h-screen flex items-center relative">
        <HeroImg />
        <div className="max-w-[700px] relative px-10 lg:px-auto m-x-auto py-10 rounded-lg">
          <h1 className="text-4xl leading-[48px] sm:text-5xl sm:leading-[58px] lg:text-6xl font-semibold lg:leading-[68px] tracking-[-2px] text-slate-50 mb-5">
            Ship Background Jobs, Crons, Webhooks, and Reliable Workflows in
            record time
          </h1>
          <p className="text-sm md:text-base text-slate-300 font-light max-w-xl leading-6 md:leading-7">
            Build, test, and deploy serverless functions driven by events or a
            schedule to any platform in seconds, with zero infrastructure.
          </p>
          <div className="flex gap-4 mt-6 lg:mt-12 items-center">
            <Link href="sign-up">
              <a className=" rounded-full text-base font-medium px-6 py-2 bg-indigo-500 transition-all text-white">
                Start Building
              </a>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
