import { useContext } from 'react';

import { DIContext } from '../contexts/di';

type Props = React.AnchorHTMLAttributes<HTMLAnchorElement>;

export function Link(props: Props) {
  const di = useContext(DIContext);
  if (!di) {
    throw new Error('missing DIContext');
  }
  return <di.Link {...props} />;
}
