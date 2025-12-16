import NextLink from 'next/link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';

type ConfigurationBlockProps = {
  icon: React.ReactNode;
  mainContent: React.ReactNode;
  subContent?: React.ReactNode;
  expression?: string;
  rightElement?: React.ReactNode;
  href?: string;
};

export default function ConfigurationBlock({
  icon,
  mainContent,
  subContent,
  expression,
  rightElement, // TODO: should rightElement be RiArrowRightSLine by default if we have href and are going to wrap in NextLink?
  href,
}: ConfigurationBlockProps) {
  const showSubContentAndExpressionSeparator = !!subContent && !!expression;

  const borderClasses =
    'border-subtle border-[0.5px] border-b-0 first:rounded-t last:rounded-b last:border-b-[0.5px]';

  const content = (
    <div className="flex items-center gap-2 self-stretch p-2">
      <div className="bg-canvasSubtle text-light flex h-9 w-9 items-center justify-center gap-2 rounded p-2">
        {icon}
      </div>
      <div className="text-basis flex min-w-0 flex-1 flex-col items-start justify-center gap-0.5 self-stretch text-sm">
        <div>{mainContent}</div>

        {(subContent || expression) && (
          <span className="text-muted flex w-full whitespace-nowrap text-sm">
            {subContent}
            {showSubContentAndExpressionSeparator && <span className="px-1">|</span>}
            {expression && (
              <Tooltip>
                <TooltipTrigger asChild className="text-muted w-full truncate text-sm">
                  <code className="font-mono">{expression}</code>
                </TooltipTrigger>
                <TooltipContent className="p-3 text-sm">
                  <code>{expression}</code>
                </TooltipContent>
              </Tooltip>
            )}
          </span>
        )}
      </div>
      <div className="self-center">{rightElement}</div>
    </div>
  );

  return href ? (
    <NextLink href={href} className={cn('hover:bg-canvasMuted block', borderClasses)}>
      {content}
    </NextLink>
  ) : (
    <div className={cn(borderClasses)}>{content}</div>
  );
}
