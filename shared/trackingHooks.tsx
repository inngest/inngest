import deterministicSplit from "deterministic-split";
import { useMemo } from "react";
import { useLocalStorage } from "react-use";

const abExperiments = {
  header: ["kill-queues-headline", "event-driven-headerline"],
  footer: ["removed", "highlighted"],
} as const;

export const useAnonId = () => {
  const [anonId] = useLocalStorage<string>("inngest-anon-id");

  return {
    anonId,
  };
};

export const useAbTest = <T extends keyof typeof abExperiments>(
  experimentName: T
) => {
  const { anonId } = useAnonId();

  const variant = useMemo(() => {
    return deterministicSplit(
      `${anonId}_${experimentName}`,
      abExperiments[experimentName]
    ) as typeof abExperiments[T][number];
  }, [anonId, experimentName]);

  return [variant];
};
