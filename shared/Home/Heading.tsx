import clsx from "clsx";

export default function Heading({
  title,
  lede,
  variant = "dark",
  className,
}: {
  title: React.ReactNode;
  lede?: React.ReactNode;
  variant?: "dark" | "light";
  className?: string;
}) {
  return (
    <div className={`tracking-tight ${className}`}>
      <h2
        className={clsx(
          "text-2xl md:text-[40px] leading-snug font-semibold",
          variant === "dark" && "text-white",
          variant === "light" && "text-slate-800"
        )}
      >
        {title}
      </h2>
      {!!lede && (
        <p
          className={clsx(
            "my-4 leading-loose text-sm md:text-base",
            variant === "dark" && "text-indigo-200",
            variant === "light" && "text-slate-500 font-medium"
          )}
        >
          {lede}
        </p>
      )}
    </div>
  );
}
