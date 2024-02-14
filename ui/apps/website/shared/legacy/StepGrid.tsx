import React, { ReactElement } from "react";
import styled from "@emotion/styled";

import Button from "./Button";

export type Step = {
  icon: ReactElement | string;
  description: string;
  action: ReactElement | string;
};

type StepGridProps = {
  steps: Step[];
};

const StepGrid = ({ steps }: StepGridProps) => {
  return (
    <Grid cols={steps.length} className="my-6">
      {steps.map((s, j) => (
        <Step key={`step-${j}`}>
          <div className="icon">
            {typeof s.icon === "string" ? (
              <img src={s.icon || "x"} alt={s.description} />
            ) : (
              s.icon
            )}
          </div>
          <div className="text">
            <span className="description">{s.description}</span>
            <span className="action">{s.action}</span>
          </div>
        </Step>
      ))}
    </Grid>
  );
};

const Grid = styled.div<{ cols: number | string }>`
  position: relative;
  display: grid;
  grid-template-columns: repeat(${({ cols }) => cols}, 1fr);
  grid-gap: ${({ cols }) => (Number(cols) > 3 ? "2rem" : "4rem")};

  &::before {
    position: absolute;
    z-index: 1;
    border-top: 2px dotted #b1a7b7;
    content: "";
    width: 100%;
    top: 50%;
  }

  @media (max-width: 900px) {
    grid-template-columns: 1fr;
    grid-gap: 1rem;

    &::before {
      border-top: none;
      border-left: 2px dotted #b1a7b7;
      width: 1px;
      top: 0px;
      left: 50%;
      height: 100%;
    }
  }
`;

const Step = styled.div`
  --spacing: 8px;

  z-index: 10;
  display: flex;
  align-items: center;
  padding: 0.8rem 1rem;
  font-size: 14px;
  background-color: var(--bg-color);
  border-radius: var(--border-radius);
  border: 2px solid var(--stroke-color);

  .icon {
    display: flex;
    justify-content: center;
    align-items: center;
    flex-shrink: 0;
    margin-right: 0.8rem;
    width: 40px;
    height: 40px;
    border-radius: var(--border-radius);
    overflow: hidden;
    text-align: center;

    img {
      max-width: 100%;
      max-height: 100%;
    }
  }

  .text {
    display: flex;
    justify-content: flex-start;
    flex-direction: column;
    gap: var(--spacing);
  }

  .description {
    font-size: 12px;
    text-transform: uppercase;
    color: var(--font-color-secondary);
  }
`;

export default StepGrid;
