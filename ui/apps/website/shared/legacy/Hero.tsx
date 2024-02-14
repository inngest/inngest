import React, { ReactElement } from 'react';
import styled from '@emotion/styled';

import Button from './Button';

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

const Hero = ({ className = '', headline, subheadline, primaryCTA, secondaryCTA }: HeroProps) => {
  return (
    <div className={className}>
      <div className="container mx-auto flex flex-row py-16 sm:py-32">
        <header className="mx-auto max-w-4xl px-6 text-center">
          <h1
            style={{ position: 'relative', zIndex: 1 }}
            className="overflow-hidden text-3xl sm:text-4xl md:text-5xl"
          >
            {headline}
          </h1>
          <p className="mx-auto max-w-xl pt-6">{subheadline}</p>
          <div className="flex flex-col justify-center gap-4 pt-6 md:flex-row">
            <Button kind="primary" size="medium" href={primaryCTA.href}>
              {primaryCTA.text}
            </Button>
            {secondaryCTA && (
              <Button kind="outline" size="medium" href={secondaryCTA.href} className="no-margin">
                {secondaryCTA.text}
              </Button>
            )}
          </div>
        </header>
      </div>
    </div>
  );
};

export default Hero;
