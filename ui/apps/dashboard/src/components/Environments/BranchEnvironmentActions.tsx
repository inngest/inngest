'use client';

import { useRouter } from 'next/navigation';
import { Button, SplitButton } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu/DropdownMenu';
import { RiArrowDownSLine } from '@remixicon/react';

import type { Environment } from '@/utils/environments';

type BranchEnvironmentActionsProps = {
  branchParent: Environment;
};

export function BranchEnvironmentActions({ branchParent }: BranchEnvironmentActionsProps) {
  const router = useRouter();

  return (
    <SplitButton
      left={
        <Button
          appearance="solid"
          className="rounded-r-none border-r-0 text-sm"
          kind="primary"
          label="Sync new app"
          href={`/env/${slugOrBranchDefault(branchParent)}/apps`}
          size="medium"
        />
      }
      right={
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              appearance="solid"
              className="ml-[1px] rounded-l-none"
              kind="primary"
              size="medium"
              icon={<RiArrowDownSLine />}
            />
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem
              className="text-basis text-sm"
              onSelect={() => router.push(`/env/${slugOrBranchDefault(branchParent)}/manage/keys`)}
            >
              Manage event keys
            </DropdownMenuItem>
            <DropdownMenuItem
              className="text-basis text-sm"
              onSelect={() =>
                router.push(`/env/${slugOrBranchDefault(branchParent)}/manage/signing-key`)
              }
            >
              Manage signing key
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      }
    />
  );
}

function slugOrBranchDefault(env: Environment) {
  return env.slug || 'branch';
}
