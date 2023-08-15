import { IconBook, IconFeed } from '../../icons';
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
    <div className="w-full h-full flex items-center justify-center text-white">
      <div className="max-w-[24rem] flex flex-col items-center justify-center text-center space-y-3">
        {imageUrl ? (
          <div className="w-48 mb-4">
            <img src={imageUrl} />
          </div>
        ) : null}

        {title ? <div className="font-semibold text-lg">{title}</div> : null}
        {subtitle ? <div>{subtitle}</div> : null}

        {link ? (
          <Link
            internalNavigation={false}
            href={link.url}
          >
            <div>{link.text}</div>
          </Link>
        ) : button ? (
          <button
            onClick={button.onClick}
            className="mt-2 bg-slate-1000/80 border border-slate-700 rounded-sm px-3 py-2 flex flex-row items-center space-x-2 justify-center leading-none"
          >
            <div>{button.text}</div>
            <IconBook />
          </button>
        ) : null}
      </div>
    </div>
  );
};
