import tailwindColors from "tailwindcss/colors";

export type IconProps = {
  size?: number;
  className?: string;
  color?: string;
};

export default function IconTheme(color: string = "transparent") {
  return {
    // NOTE - This only reads from Tailwind color constants, not any theme overrides in tailwind.config.js
    color: color === "transparent" ? "#FFFFFF" : tailwindColors[color]["500"],
    opacity: color === "transparent" ? 0.3 : 1,
  };
}
