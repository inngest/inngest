import { Badge } from '@inngest/components/Badge';
import { Card } from '@inngest/components/Card';
import { IconStar } from '@inngest/components/icons/Star';

import { Secret } from '@/components/Secret';
import { Time } from '@/components/Time';
import { DeleteSigningKeyButton } from './DeleteSigningKeyButton';

type Props = {
  signingKey: {
    createdAt: Date;
    decryptedValue: string;
    id: string;
    isActive: boolean;
    user: {
      email: string;
      name: string | null;
    } | null;
  };
};

export function SigningKey({ signingKey }: Props) {
  let accentColor = 'bg-emerald-600';
  let controls = null;
  let description = null;
  let title = 'Current key';

  if (!signingKey.isActive) {
    accentColor = 'bg-amber-400';
    controls = (
      <div className="flex gap-2">
        <DeleteSigningKeyButton signingKeyID={signingKey.id} />
      </div>
    );
    description = 'This key is inactive. You can activate it using rotation.';
    title = 'New key';
  }

  let pill = null;
  if (signingKey.createdAt > new Date(Date.now() - 24 * 60 * 60 * 1000)) {
    pill = (
      <Badge className="border-0 bg-amber-100">
        <IconStar className="text-amber-600" />
        <span className="text-amber-600">New</span>
      </Badge>
    );
  }

  return (
    <Card accentColor={accentColor} accentPosition="left" className="mb-4">
      <Card.Content className="px-4 py-0">
        <div className="py-4">
          <div className="flex">
            <div className="flex grow items-center gap-2 font-medium text-slate-950">
              {title}
              {pill}
            </div>
            {controls && <div>{controls}</div>}
          </div>
          <p className="text-sm text-slate-500">{description}</p>
        </div>

        <Secret className="mb-4" kind="signing-key" secret={signingKey.decryptedValue} />
      </Card.Content>

      <Card.Footer className="flex text-sm text-slate-500">
        <span className="grow">
          Created at <Time value={signingKey.createdAt} />
        </span>

        {signingKey.user && <span>Created by {signingKey.user.name ?? signingKey.user.email}</span>}
      </Card.Footer>
    </Card>
  );
}
