import { Button } from '@inngest/components/Button';

import Link from '@/components/Link/Link';

interface BlankSlateProps {
  imageUrl?: string;
  title?: string;
  subtitle?: string;
  link?: {
    text: string;
    url: string;
  };
  button?: { text: string; onClick: () => void };
}

export const BlankSlate = ({ imageUrl, title, subtitle, button, link }: BlankSlateProps) => {
  return (
    <div className="flex h-full w-full items-center justify-center text-white">
      <div className="flex max-w-[24rem] flex-col items-center justify-center space-y-3 text-center">
        {imageUrl ? (
          <div className="mb-4 w-48">
            <img src={imageUrl} />
          </div>
        ) : null}

        {title ? <div className="text-lg font-semibold">{title}</div> : null}
        {subtitle ? <div>{subtitle}</div> : null}

        {link ? (
          <Link href={link.url}>
            <div>{link.text}</div>
          </Link>
        ) : button ? (
          <Button btnAction={button.onClick} label={button.text} />
        ) : null}
      </div>
    </div>
  );
};
