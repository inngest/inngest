'use client';

import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import CounterInput from '@inngest/components/Forms/CounterInput';

export default function EntitlementListItemSelfServiceNumeric({
  entitlement,
  addon,
  onCancel,
  onSubmit,
}: {
  entitlement: {
    currentValue: number;
    planLimit: number;
    maxValue: number;
  };
  addon: {
    price: number; // US Cents
    quantityPer: number;
  };
  onCancel?: () => void;
  onSubmit?: (quantity: number, cost: number) => void;
}) {
  if (!onSubmit || !onCancel) {
    throw new Error('onSubmit and onCancel are required');
  }

  const startingInputValue = Math.max(entitlement.currentValue, entitlement.planLimit);
  const [inputValue, setInputValue] = useState(startingInputValue);
  const [inputValid, setInputValid] = useState(true);

  const inputQuantity = Math.ceil((inputValue - entitlement.planLimit) / addon.quantityPer);
  const cost = inputQuantity * addon.price;
  const costStr = (cost / 100).toFixed(2);

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-baseline gap-4">
        <CounterInput
          value={inputValue}
          onChange={setInputValue}
          onValid={setInputValid}
          min={entitlement.planLimit}
          max={entitlement.maxValue}
          step={addon.quantityPer}
        />
        {inputValid && <p className="text-muted text-sm">Cost: ${costStr}</p>}
      </div>
      <div className="flex items-center gap-2">
        <Button kind="secondary" appearance="ghost" onClick={onCancel} label="Cancel" />
        <Button
          appearance="outlined"
          disabled={inputValue == startingInputValue || !inputValid}
          onClick={() => {
            onSubmit(inputQuantity, cost);
          }}
          label="Update"
        />
      </div>
    </div>
  );
}
