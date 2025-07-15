'use client';

import { Button, type ButtonProps } from '@inngest/components/Button/Button';

import { useSupportContact } from '@/utils/useSupportContact';

export type AddonCardProps = {
  title: string;
  description: string;
  price?: string;
  priceUnit?: string;
  action:
    | {
        type: 'button';
        label: string;
        onClick?: () => void;
        href?: string;
        appearance?: ButtonProps['appearance'];
        kind?: ButtonProps['kind'];
      }
    | {
        type: 'input';
        buttonLabel: string;
        onAdd: (quantity: number) => void;
        defaultValue?: number;
        appearance?: ButtonProps['appearance'];
        kind?: ButtonProps['kind'];
      };
};

export function AddonCard({ title, description, price, priceUnit, action }: AddonCardProps) {
  const { contactSupport, hasContactedSupport, isReady } = useSupportContact();

  return (
    <div className="border-muted bg-canvasBase flex h-full flex-row items-center justify-between rounded-md border p-6">
      <div>
        <h4 className="text-basis mb-2 text-xl font-medium">{title}</h4>
        <p className="text-subtle mb-4">{description}</p>
        <div className="flex flex-col justify-end">
          {price && (
            <div className="mb-4 text-2xl">
              <span className="text-4xl font-medium">{price}</span>
              {priceUnit && <span className="text-subtle text-base">/{priceUnit}</span>}
            </div>
          )}
        </div>
      </div>

      {action.type === 'button' && action.label === 'Contact Sales' && (
        <Button
          onClick={contactSupport}
          appearance={action.appearance ?? 'solid'}
          kind={action.kind ?? 'primary'}
          disabled={!isReady || hasContactedSupport}
          label={hasContactedSupport ? 'Support contacted' : action.label}
        />
      )}
    </div>
  );
}
