export default function SegmentedProgressBar({
  segmentsCompleted,
  segments = 4,
}: {
  segmentsCompleted: number;
  segments?: number;
}) {
  const value = Math.round(segmentsCompleted / segments) * 100;

  return (
    <div
      className={`grid gap-1 grid-cols-${segments}`}
      role="progressbar"
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={100}
    >
      {[...Array(segments)].map((_, index) => {
        const completed = index < segmentsCompleted;
        return (
          <div
            key={index}
            className={` h-1 rounded-lg ${completed ? 'bg-btnPrimary' : ' bg-canvasMuted'}`}
          />
        );
      })}
    </div>
  );
}
