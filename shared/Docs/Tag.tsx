import clsx from "clsx";

const variantStyle = (variant: string): string => {
  switch (variant) {
    case "medium":
      return "rounded-lg px-1.5 py-1 ring-1 ring-inset";

    default:
      return null;
  }
};

const colorStyle = (color: string, variant: string): string => {
  switch (variant) {
    case "small":
      return `text-${color}-500 dark:text-${color}-400`;

    case "medium":
      return `ring-${color}-300 dark:ring-${color}-400/30 bg-${color}-400/10 text-${color}-500 dark:text-${color}-400`;

    default:
      return null;
  }
};

const valueColorMap = {
  get: "indigo",
  post: "sky",
  put: "amber",
  delete: "rose",
};

export function Tag({
  children,
  variant = "medium",
  color = valueColorMap[children.toLowerCase()] ?? "indigo",
}) {
  return (
    <span
      className={clsx(
        "font-mono text-[0.625rem] font-semibold leading-4",
        variantStyle(variant),
        colorStyle(color, variant)
      )}
    >
      {children}
    </span>
  );
}
