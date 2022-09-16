import React, { ReactElement } from "react";

import StepGrid, { Step } from "src/shared/StepGrid";

type ExamplesProps = {
  heading: ReactElement | string;
  examples: {
    title: string;
    steps: Step[];
  }[];
  cta?: {
    href: string;
    text: string;
  };
};

const Examples = ({ heading, examples = [], cta }: ExamplesProps) => {
  return (
    <div
      style={{ backgroundColor: "#f8f7fa" }}
      className="background-grid-texture"
    >
      <div className="container mx-auto max-w-5xl px-6 py-6">
        <div className="text-center px-6 max-xl mx-auto py-16">
          <h2 className="text-3xl sm:text-4xl font-normal	">{heading}</h2>
        </div>
        {examples.map((e, i) => (
          <div key={`ex-${i}`}>
            <h3 key={`title-${i}`} className="pt-6 font-normal text-xl">
              {e.title}
            </h3>
            <StepGrid steps={e.steps} />
          </div>
        ))}
        {cta && (
          <div className="text-center px-6 max-xl mx-auto py-16">
            <p className="text-2xl font-normal italic">
              <a href={cta.href}>{cta.text}</a>
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default Examples;
