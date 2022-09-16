import React, { ReactElement } from "react";
import styled from "@emotion/styled";

import Button from "src/shared/Button";

type CTA = {
  href: string;
  text: string;
};

type HeroProps = {
  className?: string;
  headline: ReactElement | string;
  subheadline: ReactElement | string;
  primaryCTA: CTA;
  secondaryCTA?: CTA;
};

const Hero = ({
  className = "",
  headline,
  subheadline,
  primaryCTA,
  secondaryCTA,
}: HeroProps) => {
  return (
    <div className={className}>
      <div className="container mx-auto py-16 sm:py-32 flex flex-row">
        <div className="text-center px-6 max-w-4xl mx-auto">
          <h1
            style={{ position: "relative", zIndex: 1 }}
            className="text-3xl sm:text-4xl md:text-5xl overflow-hidden"
          >
            {headline}
          </h1>
          <p className="pt-6 max-w-xl mx-auto">{subheadline}</p>
          <div className="flex flex flex-col gap-4 md:flex-row justify-center pt-6">
            <Button kind="primary" size="medium" href={primaryCTA.href}>
              {primaryCTA.text}
            </Button>
            {secondaryCTA && (
              <Button
                kind="outline"
                size="medium"
                href={secondaryCTA.href}
                className="no-margin"
              >
                {secondaryCTA.text}
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Hero;
