import Link from '@/components/Link/Link';
import Button from '@/components/Button';

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
          <Link href={link.url}>
            <div>{link.text}</div>
          </Link>
        ) : button ? (
          <Button
            btnAction={button.onClick}
            label={button.text}
          />
        ) : null}
      </div>
    </div>
  );
};
