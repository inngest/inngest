//
// TODO: resolveConfig is deprecated in tailwind 4 so we'll need to rework this
import resolveConfig from "tailwindcss/resolveConfig";

import { baseConfig } from "../../tailwind.config";

const resolvedConfig = resolveConfig(baseConfig);
const {
  theme: { backgroundColor, colors, textColor, placeholderColor, borderColor },
} = resolvedConfig;

export { backgroundColor, colors, textColor, placeholderColor, borderColor };
