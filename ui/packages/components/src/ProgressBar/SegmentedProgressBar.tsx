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
      aria-Valuenow={value}
      aria-Valuemin="0"
      aria-Valuemax="100"
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
