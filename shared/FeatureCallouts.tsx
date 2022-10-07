import React, { ReactElement } from "react";
import styled from "@emotion/styled";

import StepGrid, { Step } from "src/shared/StepGrid";
import Button from "src/shared/Button";

type FeatureCalloutsProps = {
  heading: ReactElement | string;
  backgrounds?: "alternating" | "gray";
  features: {
    topic: string;
    title: string;
    description: ReactElement | string;
    link?: {
      href: string;
      text?: string; // default = "Learn more"
    };
    image: ReactElement | string;
  }[];
  cta?: {
    href: string;
    text: string;
  };
};

const FeatureCallouts = ({
  heading,
  backgrounds = "alternating",
  features = [],
  cta,
}: FeatureCalloutsProps) => {
  return (
    <div className="container mx-auto max-w-5xl my-24">
      <div className="text-center px-6 max-xl mx-auto pb-16">
        <h2 className="text-4xl">{heading}</h2>
      </div>
      {features.map((f, i) => (
        <div
          key={`feature-${i}`}
          className="w-full flex flex-col lg:flex-row items-center py-4 px-8 lg:px-0"
        >
          <div
            className={`lg:w-1/2 px-6 lg:px-16 pt-8 pb-16 lg:py-0 order-2 lg:order-${
              i % 2 === 0 ? "1" : "2"
            }`}
          >
            <div className="uppercase text-color-iris-100 text-xs pb-2">
              <pre>{f.topic}</pre>
            </div>
            <h3 className="pb-2">{f.title}</h3>
            <p>{f.description}</p>
          </div>
          <div
            className={`${
              backgrounds === "gray" ? "bg-light-gray" : `alt-bg-${i}`
            } rounded-lg p-12 h-[350px] sm:h-[500px] lg:h-[400px] xl:h-[500px] w-full lg:w-1/2 bg-orange-50 order-1 lg:order-${
              i % 2 === 0 ? "2" : "1"
            } flex items-center justify-center`}
          >
            {typeof f.image === "string" ? (
              <img src={f.image} alt={`A graphic of ${f.title} feature`} />
            ) : (
              f.image
            )}
          </div>
        </div>
      ))}
      {cta && (
        <div className="text-center px-6 max-xl mx-auto pt-16 flex flex justify-center">
          <Button kind="outlinePrimary" href={cta.href}>
            {cta.text}
          </Button>
        </div>
      )}
    </div>
  );
};

export default FeatureCallouts;
