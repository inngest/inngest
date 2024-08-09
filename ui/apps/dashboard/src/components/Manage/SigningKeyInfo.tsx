import { Link } from '@inngest/components/Link/Link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip/Tooltip';
import { RiQuestionLine } from '@remixicon/react';

export const SigningKeyInfo = () => (
  <Tooltip>
    <TooltipTrigger>
      <RiQuestionLine className="text-muted h-[18px] w-[18px]" />
    </TooltipTrigger>
    <TooltipContent
      side="right"
      sideOffset={2}
      className="border-muted text-subtle text-md mt-6 flex flex-col rounded-lg border p-0"
    >
      <div className="border-b px-4 py-2 ">
        Use the secret signing key with the Inngest SDK to enable us securely communicate with your
        application.
      </div>

      <div className="px-4 py-2">
        <Link href={'https://www.inngest.com/docs/platform/signing-keys'} className="text-md">
          Learn how create a webhook
        </Link>
      </div>
    </TooltipContent>
  </Tooltip>
);
