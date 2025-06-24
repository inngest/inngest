import resolveConfig from 'tailwindcss/resolveConfig';

import tailwindConfig from '../../tailwind.config';
import { resolveColor } from './colors';

const {
  theme: { backgroundColor, textColor },
} = resolveConfig(tailwindConfig);

export const jsonTreeTheme = (dark: boolean): Record<string, any> => ({
  base00: resolveColor(backgroundColor.codeEditor, dark),
  base0D: resolveColor(textColor.codeStringKeyJson, dark),
  base09: resolveColor(textColor.codeNumberJson, dark),
  tree: {
    border: 0,
    paddingLeft: 6,
    paddingBottom: 6,
    paddingTop: 0,
    marginTop: 0,
    marginBottom: 0,
    marginLeft: 0,
    marginRight: 0,
    listStyle: 'none',
    MozUserSelect: 'none',
    WebkitUserSelect: 'none',
  },
  arrow: ({ style }: { style: any }, nodeType: any, expanded: boolean) => ({
    style: {
      ...style,
      marginLeft: 4,
      marginRight: 4,
      transition: '150ms',
      WebkitTransition: '150ms',
      MozTransition: '150ms',
      transformOrigin: '45% 50%',
      WebkitTransformOrigin: '45% 50%',
      MozTransformOrigin: '45% 50%',
      position: 'relative',
      lineHeight: '.8em',
      fontSize: '0.75em',
      display: 'inline-block',
      width: '8px',
      height: '8px',
      borderRight: '1.5px solid currentColor',
      borderBottom: '1.5px solid currentColor',
      transform: expanded ? 'rotateZ(45deg)' : 'rotateZ(-45deg)',
      WebkitTransform: expanded ? 'rotateZ(45deg)' : 'rotateZ(-45deg)',
      MozTransform: expanded ? 'rotateZ(45deg)' : 'rotateZ(-45deg)',
      textIndent: '-9999px',
    },
  }),
});
