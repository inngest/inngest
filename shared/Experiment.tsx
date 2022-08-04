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
  if (isSsr) return null;

  return <>{props.variants[variant]}</>;
};
