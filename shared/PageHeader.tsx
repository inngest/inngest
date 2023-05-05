import Image from "next/image";

import { Button } from "./Button";

type PageHeaderProps = {
  title: string;
  lede: string;
  image?: string;
  ctas?: {
    href: string;
    text: string;
    arrow?: "left" | "right";
  }[];
};

export default function PageHeader({
  title,
  lede,
  image,
  ctas = [],
}: PageHeaderProps) {
  return (
    <div className="py-24 md:py-48 flex flex-col lg:flex-row gap-8 justify-between lg:items-center">
      <div className="lg:w-7/12 max-w-2xl">
        <h1 className="text-4xl leading-[48px] sm:text-5xl sm:leading-[58px] lg:text-6xl font-semibold lg:leading-[68px] tracking-[-2px] text-slate-50 mb-5">
          {title}
        </h1>
        <p
          className="text-sm md:text-base text-slate-200 max-w-xl leading-6 md:leading-7"
          dangerouslySetInnerHTML={{ __html: lede }}
        ></p>
        {Boolean(ctas.length) && (
          <div className="mt-5">
            {ctas.map((cta) => (
              <Button {...cta}>{cta.text}</Button>
            ))}
          </div>
        )}
      </div>
      {Boolean(image) && (
        <div className="shrink w-[75%] max-w-lg lg:w-5/12 h-auto mx-auto">
          <Image
            src={image}
            width="720"
            height="360"
            alt={`Hero image for ${title}`}
          />
        </div>
      )}
    </div>
  );
}
