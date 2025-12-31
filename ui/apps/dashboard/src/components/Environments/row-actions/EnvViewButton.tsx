import { useNavigate } from '@tanstack/react-router';
import { Button } from '@inngest/components/Button';

import { type Environment } from '@/utils/environments';
import { pathCreator } from '@/utils/urls';

type Props = {
  env: Pick<Environment, 'slug'>;
};

export function EnvViewButton({ env }: Props) {
  const navigate = useNavigate();

  return (
    <Button
      appearance="outlined"
      kind="secondary"
      label="View"
      size="small"
      onClick={() => navigate({ to: pathCreator.apps({ envSlug: env.slug }) })}
    />
  );
}
