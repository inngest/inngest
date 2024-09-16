export default function StepsPageHeader({
  currentStep,
  totalSteps,
  title,
}: {
  currentStep: number;
  totalSteps: number;
  title: string;
}) {
  return (
    <>
      <p className="text-light mb-1 text-xs font-medium uppercase">
        Step {currentStep} of {totalSteps}
      </p>
      <h2 className="text-basis mb-4 text-xl">{title}</h2>
    </>
  );
}
