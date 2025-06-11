import NextLink from 'next/link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';

type ConfigurationBlockProps = {
  icon: React.ReactNode;
  mainContent: string;
  subContent?: string;
  expression?: string;
  rightElement?: React.ReactNode;
  href?: string;
};

export default function ConfigurationBlock({
  icon,
  mainContent,
  subContent,
  expression,
  rightElement,
  href,
}: ConfigurationBlockProps) {
  const showSubContentAndExpressionSeparator = !!subContent && !!expression;

  const content = (
    <div className="border-subtle flex items-center gap-2 self-stretch border border-b-0 p-2 first:rounded-t last:rounded-b last:border-b">
      <div className="bg-canvasSubtle text-light flex h-9 w-9 items-center justify-center gap-2 rounded p-2">
        {icon}
      </div>
      <div className="text-basis flex min-w-0 flex-col items-start justify-center gap-1 self-stretch text-sm font-medium">
        <div>{mainContent}</div>

        <span className="flex w-full whitespace-nowrap">
          {subContent}
          {showSubContentAndExpressionSeparator && <span className="px-1">|</span>}
          {expression && (
            <Tooltip>
              <TooltipTrigger asChild className="text-muted w-full truncate text-sm">
                <code className="font-mono">{expression}</code>
              </TooltipTrigger>
              <TooltipContent className="text-muted bg-canvasBase p-3 text-sm">
                <div>
                  <h2 className="text-basis gap-1 text-xs">Expression</h2>
                  <div className="bg-codeEditor border-subtle text-muted flex items-start gap-2 self-stretch border px-3 text-xs leading-5">
                    {expression}
                  </div>
                </div>
              </TooltipContent>
            </Tooltip>
          )}
        </span>
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
