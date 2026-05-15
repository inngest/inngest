import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { RiErrorWarningLine } from '@remixicon/react';

export function ScoreCalculationExplainer() {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="How is the score calculated?"
          className="text-subtle hover:text-basis flex items-center"
        >
          <RiErrorWarningLine className="h-[14px] w-[14px]" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[480px] max-w-[90vw]">
        <div className="flex flex-col gap-3 p-3">
          <div>
            <h4 className="text-basis text-sm font-medium">
              How is this calculated?
            </h4>
            <p className="text-muted mt-1 text-xs">
              For each enabled metric we normalize the variant&apos;s average to
              its min/max range (inverting when configured) and weight by the
              points you allocated.
            </p>
          </div>

          <div className="bg-canvasSubtle text-basis font-serif rounded p-4">
            <ScoringEquation />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function ScoringEquation() {
  return (
    <div className="grid grid-cols-[auto_auto_1fr] items-center gap-x-2 gap-y-4 text-sm">
      <Var className="justify-self-end">contribution</Var>
      <span>=</span>
      <div className="flex items-center gap-1.5">
        <Var>points</Var>
        <span>×</span>
        <span className="whitespace-nowrap">min(max(</span>
        <Fraction
          numerator={
            <>
              <Var>avg</Var> − <Var>min</Var>
            </>
          }
          denominator={
            <>
              <Var>max</Var> − <Var>min</Var>
            </>
          }
        />
        <span className="whitespace-nowrap">, 0), 1)</span>
      </div>

      <Var className="justify-self-end">total</Var>
      <span>=</span>
      <div className="flex items-center gap-1.5">
        <span className="text-lg leading-none">Σ</span>
        <Var>contribution</Var>
      </div>
    </div>
  );
}

function Var({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <span className={`italic ${className ?? ''}`}>{children}</span>;
}

function Fraction({
  numerator,
  denominator,
}: {
  numerator: React.ReactNode;
  denominator: React.ReactNode;
}) {
  return (
    <span className="inline-flex flex-col items-center px-1 align-middle text-center leading-tight">
      <span className="whitespace-nowrap border-b border-current px-1.5 pb-0.5">
        {numerator}
      </span>
      <span className="whitespace-nowrap px-1.5 pt-0.5">{denominator}</span>
    </span>
  );
}
