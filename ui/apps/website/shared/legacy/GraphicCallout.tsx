import React, { ReactElement } from 'react';

import Button from './Button';

type GraphicCalloutProps = {
  heading: ReactElement | string;
  description: ReactElement | string;
  image: string;
  cta?: {
    href: string;
    text: string;
  };
  style?: React.CSSProperties;
};

const GraphicCallout = ({
  heading,
  description,
  image,
  cta,
  style = {
    backgroundColor: 'var(--color-almost-black)',
    color: 'var(--color-white)',
  },
}: GraphicCalloutProps) => {
  return (
    <div style={style} className="bg-texture-gridlines-30 my-16">
      <div className="flex flex-col justify-items-end gap-8 md:flex-row">
        <div className="lg:pl-10p mx-12 flex basis-2/5 flex-col justify-center pt-12 md:pb-12 lg:mx-0">
          <h2 className="text-3xl font-normal">{heading}</h2>
          <p className="my-6">{description}</p>
          {cta && (
            <div>
              <Button
                href={cta.href}
                kind="outline"
                size="medium"
                style={{ display: 'inline-flex' }}
              >
                {cta.text}
              </Button>
            </div>
          )}
        </div>
        <div className="flex basis-3/5 flex-col justify-end">
          <img src={image} />
        </div>
      </div>
    </div>
  );
};

export default GraphicCallout;
