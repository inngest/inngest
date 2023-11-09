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
    <div className={`${className}`}>
      <h2
        className={clsx(
          "text-2xl md:text-5xl leading-snug font-semibold tracking-tight ",
          variant === "dark" &&
            "bg-gradient-to-br from-white to-slate-300 bg-clip-text text-transparent",
          variant === "light" && "text-slate-800"
        )}
      >
        {title}
      </h2>
      {!!lede && (
        <p
          className={clsx(
            "my-4 font-medium leading-loose text-md md:text-lg",
            variant === "dark" && "text-indigo-100/90",
            variant === "light" && "text-slate-500 font-medium"
          )}
        >
          {lede}
        </p>
      )}
    </div>
  );
}
