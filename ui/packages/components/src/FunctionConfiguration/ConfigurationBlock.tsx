import NextLink from 'next/link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';

type ConfigurationBlockProps = {
  icon: React.ReactNode;
  mainText: string;
  subText?: React.ReactNode;
  rightElement?: React.ReactNode;
  href?: string;
};

export default function ConfigurationBlock({
  icon,
  mainText,
  subText,
  rightElement,
  href,
}: ConfigurationBlockProps) {
  const content = (
    <div className="border-subtle flex items-center gap-2 self-stretch border border-b-0 p-2 first:rounded-t last:rounded-b last:border-b">
      <div className="bg-canvasSubtle text-light flex h-9 w-9 items-center justify-center gap-2 rounded p-2">
        {icon}
      </div>
      <div className="text-basis flex min-w-0 flex-col items-start justify-center gap-1 self-stretch text-sm font-medium">
        <div>{mainText}</div>

        {subText && (
          <Tooltip>
            <TooltipTrigger asChild className="text-muted w-full truncate text-sm">
              {subText}
            </TooltipTrigger>
            <TooltipContent className="text-muted bg-canvasBase p-3 text-sm">
              <div>
                <h2 className="text-basis gap-1 text-xs">Expression</h2>
                <div className="bg-codeEditor border-subtle text-muted flex items-start gap-2 self-stretch border px-3 text-xs leading-5">
                  {subText}
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        )}
        {/*{subText && <div className="text-muted truncate text-sm">{subText}</div>}*/}
      </div>
      {/*TODO should this be RiArrowRightSLine by default for NextLink*/}
      {rightElement}
    </div>
  );

  return href ? (
    <NextLink
      href={href}
      className="border-subtle bg-canvasBase hover:bg-canvasMuted block rounded-md border"
    >
      {content}
    </NextLink>
  ) : (
    content
  );
}
