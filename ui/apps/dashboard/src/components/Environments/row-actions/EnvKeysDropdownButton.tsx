import { useNavigate } from '@tanstack/react-router';
import { Button } from '@inngest/components/Button/NewButton';
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
  const navigate = useNavigate();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          appearance="outlined"
          icon={<RiKey2Line />}
          kind="secondary"
          size="small"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem
          className="text-basis text-sm"
          onSelect={() =>
            navigate({ to: pathCreator.keys({ envSlug: env.slug }) })
          }
        >
          Manage event keys
        </DropdownMenuItem>
        <DropdownMenuItem
          className="text-basis text-sm"
          onSelect={() =>
            navigate({ to: pathCreator.signingKeys({ envSlug: env.slug }) })
          }
        >
          Manage signing key
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
