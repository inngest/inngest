import React, { useEffect } from "react";
import { abExperiments, useAbTest } from "./trackingHooks";

interface ExperimentProps<T extends keyof typeof abExperiments> {
  /**
   * The experiment to use.
   */
  experiment: T;

  /**
   * Children to render for each variant of the chosen experiment. All variants
   * must be accounted for.
   */
  variants: Record<typeof abExperiments[T][number], React.ReactNode>;

  /**
   * Images should not be rendered on the server as React won't update the attributes correctly
   */
  isImage?: boolean;
}

/**
 * Render a component based on the user's current variant of an experiment.
 *
 * Requires client-side logic for understanding which variant the user should
 * see, so SSR will return `null` and the component will be rendered once the
 * view is hydrated on the client.
 */
export const Experiment = <T extends keyof typeof abExperiments>(
  props: ExperimentProps<T>
) => {
  const { variant } = useAbTest(props.experiment);

  const isSsr = typeof window === "undefined";
  if (isSsr && props.isImage) return null;

  return <>{props.variants[variant]}</>;
};

/**
 * FadeIn a nested component quickly after loading the page
 * to prevent a flash of an experiment variation that the user
 * isn't meant to see. This should wrap the parent components
 * where <Experiment> is used.
 */
export const FadeIn = ({ children }) => {
  const [isVisible, setVisible] = React.useState(false);
  useEffect(() => {
    const timer = setTimeout(() => {
      setVisible(true);
    }, 50);
    return () => clearTimeout(timer);
  }, []);
  return (
    <div
      style={{
        transition: "opacity 100ms ease-in",
        opacity: isVisible ? 1 : 0,
      }}
    >
      {children}
    </div>
  );
};
