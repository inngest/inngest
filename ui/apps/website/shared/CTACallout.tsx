import { Button } from 'src/shared/Button';

type CTACalloutProps = {
  text: React.ReactNode | string;
  cta?: {
    href: string;
    text: string;
  };
  wide?: boolean;
};

export default function CTACallout({ text, cta, wide = false }: CTACalloutProps) {
  return (
    <aside
      className={`not-prose ${
        wide ? 'max-w-[80ch]' : 'max-w-[70ch]'
      } m-auto my-12 flex flex-col items-start gap-4 rounded-lg border border-indigo-900/50 bg-indigo-900/20 px-6  py-5 leading-relaxed text-indigo-100`}
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
