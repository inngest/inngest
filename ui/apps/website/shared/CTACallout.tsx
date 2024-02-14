import { Button } from "src/shared/Button";

type CTACalloutProps = {
  text: React.ReactNode | string;
  cta?: {
    href: string;
    text: string;
  };
  wide?: boolean;
};

export default function CTACallout({
  text,
  cta,
  wide = false,
}: CTACalloutProps) {
  return (
    <aside
      className={`not-prose ${
        wide ? "max-w-[80ch]" : "max-w-[70ch]"
      } m-auto bg-indigo-900/20 text-indigo-100 flex flex-col items-start gap-4 leading-relaxed rounded-lg py-5 px-6  my-12 border border-indigo-900/50`}
    >
      <p className="text-sm lg:text-base">{text}</p>
      {!!cta && (
        <Button href={cta.href} arrow="right">
          {cta.text}
        </Button>
      )}
    </aside>
  );
}
