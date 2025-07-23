'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { RiKey2Line } from '@remixicon/react';

import { type Environment } from '@/utils/environments';
import { pathCreator } from '@/utils/urls';

type Props = {
  env: Pick<Environment, 'slug'>;
};

export function EnvKeysDropdownButton({ env }: Props) {
  const router = useRouter();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button appearance="outlined" icon={<RiKey2Line />} kind="secondary" size="small" />
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem
          className="text-basis text-sm"
          onSelect={() => router.push(pathCreator.keys({ envSlug: env.slug }))}
        >
          Manage event keys
        </DropdownMenuItem>
        <DropdownMenuItem
          className="text-basis text-sm"
          onSelect={() => router.push(pathCreator.signingKeys({ envSlug: env.slug }))}
        >
          Manage signing key
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
