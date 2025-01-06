import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link/Link';
import { RiExternalLinkLine } from '@remixicon/react';

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

export function BlankSlate({ imageUrl, title, subtitle, button, link }: BlankSlateProps) {
  return (
    <div className="text-basis flex h-full w-full items-center justify-center">
      <div className="flex max-w-[24rem] flex-col items-center justify-center space-y-3 text-center">
        {imageUrl ? (
          <div className="mb-4 w-48">
            <img src={imageUrl} />
          </div>
        ) : null}

        {title ? <div className="text-lg">{title}</div> : null}
        {subtitle ? <div>{subtitle}</div> : null}

        {link ? (
          <Link href={link.url} size="small" iconAfter={<RiExternalLinkLine className="h-4 w-4" />}>
            <div>{link.text}</div>
          </Link>
        ) : button ? (
          <Button onClick={button.onClick} label={button.text} />
        ) : null}
      </div>
    </div>
  );
}
