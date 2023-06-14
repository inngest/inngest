import Image from "next/image";
import { Button } from "src/shared/Button";

type CustomerQuoteProps = {
  className?: string;
  logo: string;
  text: React.ReactNode | string;
  cta: {
    href: string;
    text: string;
  };
};

const MeshGradient = `
radial-gradient(at 21% 4%, hsla(209,94%,39%,0.69) 0px, transparent 50%),
radial-gradient(at 97% 96%, hsla(279,100%,76%,0.53) 0px, transparent 50%),
radial-gradient(at 43% 61%, hsla(279,100%,76%,0.53) 0px, transparent 50%),
radial-gradient(at 73% 24%, hsla(279,100%,76%,0.7) 0px, transparent 50%),
radial-gradient(at 5% 94%, hsla(210,75%,64%,0.81) 0px, transparent 50%).
`;

export default function CustomerQuote({
  className = "",
  logo,
  text,
  cta,
}: CustomerQuoteProps) {
  return (
    <aside
      className={`${className} max-w-[1200px] mx-auto px-4 flex flex-col md:flex-row items-center`}
    >
      <div className="relative max-w-[80%]">
        <div className="absolute z-0 w-full h-full rounded-[14px] backdrop-blur bg-white/5"></div>
        <div
          className="relative z-10 m-5 py-5 px-14 h-36 md:h-96 md:w-72 flex items-center rounded"
          style={{
            backgroundColor: `hsla(235,79%,63%,1)`,
            backgroundImage: MeshGradient,
          }}
        >
          <img
            src={logo}
            alt={`Customer logo`}
            className="w-[100%] md:max-w-full"
          />
        </div>
      </div>
      <div className="p-5 -mt-8 md:mt-0 md:-ml-16 border border-slate-600/30 rounded-[24px]">
        <div className="p-6 pt-8 md:py-10 md:pl-24 md:pr-16 flex flex-col items-start	gap-6 rounded-[10px] text-white bg-slate-900">
          <p className="text-lg lg:text-xl">&ldquo;{text}&rdquo;</p>
          <Button href={cta.href} variant="secondary" arrow="right">
            {cta.text}
          </Button>
        </div>
      </div>
    </aside>
  );
}
