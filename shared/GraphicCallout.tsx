import React, { ReactElement } from "react";

import Button from "src/shared/Button";

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
    backgroundColor: "var(--color-almost-black)",
    color: "var(--color-white)",
  },
}: GraphicCalloutProps) => {
  return (
    <div style={style} className="background-grid-texture my-16">
      <div className="flex flex-col md:flex-row justify-items-end gap-8">
        <div className="basis-2/5 pt-12 md:pb-12 mx-12 lg:mx-0 lg:pl-10p flex flex-col justify-center">
          <h2 className="text-3xl font-normal">{heading}</h2>
          <p className="my-6">{description}</p>
          {cta && (
            <div>
              <Button
                href={cta.href}
                kind="outline"
                size="medium"
                style={{ display: "inline-flex" }}
              >
                {cta.text}
              </Button>
            </div>
          )}
        </div>
        <div className="basis-3/5 flex flex-col justify-end">
          <img src={image} />
        </div>
      </div>
    </div>
  );
};

export default GraphicCallout;
