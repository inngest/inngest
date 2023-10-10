import React, { useState } from "react";
import classNames from "src/utils/classNames";
import { Button } from "../Button";
import InformationCircle from "src/shared/Icons/InformationCircle";
import * as SliderPrimitive from "@radix-ui/react-slider";
import { type Plan } from "../../pages/pricing";

export default function PlanCard({
  variant = "light",
  content,
}: {
  variant: string;
  content: Plan;
}) {
  const [stepCalculator, setStepCalculator] = useState<number>(
    typeof content.cost.included === "number" ? content.cost.included : 0
  );
  // Calculate price with additional steps from calcuator
  const additionalCost =
    typeof content.cost.included === "number" &&
    typeof content.cost.additionalPrice === "number"
      ? ((stepCalculator - content.cost.included) /
          content.cost.additionalRate) *
        content.cost.additionalPrice
      : 0;
  const price =
    typeof content.cost.basePrice === "number"
      ? content.cost.basePrice + additionalCost
      : content.cost.basePrice;

  const theme = {
    light: {
      cardBG: "bg-slate-100",
      price: "text-indigo-500",
      row: "odd:bg-slate-400/10",
      primary: "text-slate-800",
      secondary: "text-slate-600",
      description: "text-slate-600",
    },
    focus: {
      cardBG: "bg-white",
      price: "text-indigo-500",
      row: "odd:bg-slate-400/10",
      primary: "text-slate-800",
      secondary: "text-slate-600",
      description: "text-slate-600",
    },
    dark: {
      cardBG: "bg-slate-900/90",
      price: "text-indigo-400",
      row: "odd:bg-slate-400/10",
      primary: "text-white",
      secondary: "text-slate-400",
      description: "text-slate-200",
    },
  };

  return (
    <div
      className={`w-full rounded-lg md:rounded-l-none md:rounded-r-none md:first:rounded-l-lg md:last:rounded-r-lg flex flex-col justify-between text-center ${theme[variant].cardBG}`}
    >
      <div className="pt-8">
        {content.popular && (
          <div className="-mt-11 mb-3.5 block">
            <div className="bg-indigo-500 inline-block shadow-lg rounded-full text-white text-sm font-semibold tracking-tight leading-none py-2 px-4">
              Most popular
            </div>
          </div>
        )}
        {/* Prevent weird button wrap on enterprise from mis-aligning rows */}
        <div className="sm:min-h-[272px] min-[933px]:min-h-[252px] min-[1272px]:min-h-0">
          <h2 className={`text-lg font-semibold ${theme[variant].primary}`}>
            {content.name}
          </h2>

          <p
            className={`text-4xl mt-4 font-bold tracking-tight text-indigo-500 ${
              theme[variant].price
            } ${!!content.cost.period && "pl-4"}`}
          >
            ${price}
            <span
              className={`text-sm font-medium ml-0.5 ${theme[variant].secondary}`}
            >
              {!!content.cost.period ? `/${content.cost.period}` : ""}
            </span>
          </p>

          <div className="px-12 py-2 mt-4 mb-4">
            {/*  */}
            <Button
              href={content.cta.href}
              arrow="right"
              full
              variant="primary"
            >
              {content.cta.text}
            </Button>
          </div>

          <p
            className={`text-base mt-4 font-medium flex items-center justify-center ${theme[variant].description}`}
          >
            {typeof content.cost.included === "string"
              ? content.cost.included
              : stepCalculator?.toLocaleString(undefined, {
                  notation: "compact",
                  compactDisplay: "short",
                })}{" "}
            steps{" "}
            <a
              href="#what-is-a-function-step"
              className="ml-1.5 transition-all text-slate-500 hover:text-slate-700"
            >
              <InformationCircle size="1.2em" />
            </a>
          </p>

          <div className="px-24 my-4 h-4">
            {/* Set height to align columns */}
            {typeof content.cost.included === "number" &&
            content.cost.additionalPrice ? (
              <Slider
                defaultValue={[content.cost.included]}
                min={content.cost.included}
                max={content.cost.max}
                step={content.cost.additionalRate}
                onValueChange={(v) => setStepCalculator(v[0])}
              />
            ) : (
              <div></div>
            )}
          </div>

          <div className="flex justify-center mt-4">
            <div className={`text-sm font-medium ${theme[variant].secondary}`}>
              {content.cost.additionalPrice ? (
                <>
                  + <strong>{content.cost.additionalPrice}</strong> per
                  additional{" "}
                  <strong>
                    {content.cost.additionalRate.toLocaleString(undefined, {
                      notation: "compact",
                      compactDisplay: "short",
                    })}
                  </strong>
                </>
              ) : (
                <>&nbsp;</>
              )}
            </div>
          </div>
        </div>
        {/* <p
          className={`text-sm mt-2 font-medium  ${theme[variant].description}`}
        >
          {content.description}
        </p> */}
        <ul className="flex flex-col mt-6">
          {content.features.map((feature, i) => (
            <li
              key={i}
              className={`flex flex-col py-2.5 min-h-[64px] ${theme[variant].row}`}
            >
              {feature.quantity && (
                <span className={`font-semibold ${theme[variant].primary}`}>
                  {feature.quantity}
                </span>
              )}
              <span
                className={classNames(
                  feature.quantity
                    ? `font-medium text-sm ${theme[variant].secondary}`
                    : `font-semibold my-2 ${theme[variant].primary}`,
                  `  tracking-tight`
                )}
              >
                {feature.text}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

const Slider = React.forwardRef<
  React.ElementRef<typeof SliderPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SliderPrimitive.Root>
>(({ className, ...props }, ref) => (
  <SliderPrimitive.Root
    ref={ref}
    className={classNames(
      "relative flex w-full touch-none select-none items-center",
      className
    )}
    {...props}
  >
    <SliderPrimitive.Track className="relative h-1 w-full grow overflow-hidden rounded-full bg-slate-200">
      <SliderPrimitive.Range className="absolute h-full bg-slate-600" />
    </SliderPrimitive.Track>
    <SliderPrimitive.Thumb className="block h-5 w-5 rounded-full border-2 border-primary bg-slate-600 ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50" />
  </SliderPrimitive.Root>
));
Slider.displayName = SliderPrimitive.Root.displayName;
