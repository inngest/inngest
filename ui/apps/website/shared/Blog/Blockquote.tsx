import { Button } from 'src/shared/Button';

type BlockquoteProps = {
  text: React.ReactNode | string;
  attribution: {
    name: string;
    title: string;
  };
  avatar?: string;
};

export default function Blockquote({ text, attribution, avatar }: BlockquoteProps) {
  return (
    <figure className="not-prose rounded-lg border border-indigo-300/20 py-6 pl-8 pr-10">
      <blockquote className="text-lg">&ldquo;{text}&rdquo;</blockquote>
      <div className="mt-6 flex flex-row items-center gap-4">
        {!!avatar && (
          <img
            className="h-10 w-10 rounded-full"
            src={avatar}
            alt={`Image of ${attribution.name}`}
          />
        )}
        <figcaption>
          <span className="font-semibold text-white">{attribution.name}</span> - {attribution.title}
        </figcaption>
      </div>
    </figure>
  );
}
