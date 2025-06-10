import NextLink from 'next/link';

type ConfigurationBlockProps = {
  key?: React.Key;
  icon: React.ReactNode;
  mainText: string;
  subText?: React.ReactNode;
  rightElement?: React.ReactNode;
  href?: string;
};

export default function ConfigurationBlock({
  key,
  icon,
  mainText,
  subText,
  rightElement,
  href,
}: ConfigurationBlockProps) {
  const content = (
    <div
      key={key}
      className="border-subtle flex items-center gap-2 self-stretch border border-b-0 p-2 first:rounded-t last:rounded-b last:border-b"
    >
      <div className="bg-canvasSubtle text-light flex h-9 w-9 items-center justify-center gap-2 rounded p-2">
        {icon}
      </div>
      <div className="text-basis flex grow flex-col items-start justify-center gap-1 self-stretch text-sm font-medium">
        <div>{mainText}</div>
        {subText && <div className="text-muted text-sm">{subText}</div>}
      </div>
      {rightElement}
    </div>
  );

  return href ? (
    <NextLink
      href={href}
      className="border-subtle bg-canvasBase hover:bg-canvasMuted block rounded-md border border-gray-200 "
    >
      {content}
    </NextLink>
  ) : (
    content
  );
}
