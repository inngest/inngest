import resolveConfig from "tailwindcss/resolveConfig";
import tailwindConfig from "tailwind.config.js";

const twConfig = resolveConfig(tailwindConfig);


export default function IconTheme(color: string = 'transparent') {

  return {
    color: color === 'transparent' ? '#FFFFFF' : twConfig.theme.colors[color]['500'],
    opacity: color === 'transparent' ? 0.3 : 1,
  }

}