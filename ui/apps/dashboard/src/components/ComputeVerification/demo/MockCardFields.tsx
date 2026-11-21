import { Input } from '@inngest/components/Forms/Input';
import type { CardFieldsProps } from '../types';

/** Fixed, read-only fixtures: the demo never accepts card data. */
export function MockCardFields({
  disabled,
  errorID,
  populated,
}: CardFieldsProps & { populated: boolean }) {
  const field = (
    id: string,
    label: string,
    placeholder: string,
    value: string,
  ) => (
    <div className="flex min-w-0 flex-col gap-1">
      <label htmlFor={id} className="text-basis text-sm">
        {label}
      </label>
      <Input
        id={id}
        placeholder={placeholder}
        value={populated ? value : ''}
        readOnly
        disabled={disabled}
        className="h-10 cursor-default border-muted bg-canvasBase text-basis disabled:opacity-50"
      />
    </div>
  );
  return (
    <div
      className="flex flex-col gap-3"
      role="group"
      aria-label="Card details"
      aria-describedby={errorID}
    >
      {field(
        'mock-card-name',
        'Name on card',
        'Name as it appears on your card',
        'Sanjana Laddha',
      )}
      {field(
        'mock-card-number',
        'Card number',
        '•••• •••• •••• ••••',
        '4242 4242 4242 4242',
      )}
      <div className="grid grid-cols-2 gap-3">
        {field('mock-card-expiry', 'Expiry', 'MM / YY', '12 / 29')}
        {field('mock-card-cvc', 'CVC', '•••', '•••')}
      </div>
      {field('mock-card-zip', 'ZIP code', '12345', '94107')}
    </div>
  );
}
