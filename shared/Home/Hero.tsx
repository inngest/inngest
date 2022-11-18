import Link from "next/link";
import HeroImg from "./HomeImg/HeroImg";

export default function Hero() {
  return (
    <div className="relative">
      <div
        style={{
          background: "radial-gradient(circle at center, #13123B, #08090d)",
        }}
        className="absolute w-[200vw] -z-10 -translate-x-1/2 -translate-y-1/2 h-[200vw] rounded-full blur-lg opacity-90"
      ></div>

      <div className="max-w-container-desktop m-auto px-10 py-96 max-h-screen flex items-center relative">
        <HeroImg />
        <div className="max-w-[700px]">
          <h1 className="text-7xl font-semibold tracking-[-2px] leading-[85px] text-slate-50 mb-5">
            Ship event-driven code in record time
          </h1>
          <p className="text-slate-300 font-light max-w-xl leading-7">
            Build, test, and deploy serverless functions driven by events or a
            schedule to any platform in seconds, with zero infrastructure.
          </p>
          <div className="flex gap-4 mt-12 items-center">
            <code className="text-xs mr-5">
              <span className="text-slate-500 mr-2">$</span>
              npm install inngest
            </code>
            <Link
              className="border border-slate-800 rounded-full text-base px-6 py-2 hover:bg-slate-800/50 transition-all hover:text-white"
              href="sign-up"
            >
              Start Building
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
