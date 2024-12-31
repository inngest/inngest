import { Card } from '@inngest/components/Card';
import { Pill } from '@inngest/components/Pill';
import { Time } from '@inngest/components/Time';
import { RiStarFill } from '@remixicon/react';

import { Secret } from '@/components/Secret';
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
  let accentColor = 'bg-primary-moderate';
  let controls = null;
  let description = null;
  let title = 'Current key';

  if (!signingKey.isActive) {
    accentColor = 'bg-accent-moderate';
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
      <Pill kind="warning">
        <RiStarFill size={16} className="pr-1" />
        <span>New</span>
      </Pill>
    );
  }

  return (
    <Card accentColor={accentColor} accentPosition="left" className="mb-4">
      <Card.Content className="px-4 py-0">
        <div className="py-4">
          <div className="flex">
            <div className="flex grow items-center gap-2 font-medium">
              {title}
              {pill}
            </div>
            {controls && <div>{controls}</div>}
          </div>
          <p className="text-subtle text-sm">{description}</p>
        </div>

        <Secret className="mb-4" kind="signing-key" secret={signingKey.decryptedValue} />
      </Card.Content>

      <Card.Footer className="text-subtle flex text-sm">
        <span className="grow">
          Created at <Time value={signingKey.createdAt} />
        </span>

        {signingKey.user && <span>Created by {signingKey.user.name ?? signingKey.user.email}</span>}
      </Card.Footer>
    </Card>
  );
}
