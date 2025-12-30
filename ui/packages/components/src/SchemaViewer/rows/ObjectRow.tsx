import { cn } from '../../utils/classNames';
import { useExpansion } from '../ExpansionContext';
import type { ObjectNode, SchemaNode, ValueNode } from '../types';
import { CollapsibleRowWidget } from './CollapsibleRowWidget';
import { Row } from './Row';
import { ValueRow } from './ValueRow';

export type ObjectRowProps = { node: ObjectNode; typeLabelOverride?: string };

export function ObjectRow({ node, typeLabelOverride }: ObjectRowProps): React.ReactElement {
  const { isExpanded, toggle } = useExpansion();

  const open = isExpanded(node.path);

  return (
    <div className="flex flex-col gap-1">
      <div className="flex cursor-pointer items-center" onClick={() => toggle(node.path)}>
        <CollapsibleRowWidget open={open} />
        <div className="-ml-0.5">
          <ValueRow
            boldName={open}
            node={makeFauxValueNode(node)}
            typeLabelOverride={typeLabelOverride ?? ''}
          />
        </div>
      </div>
      {open && (
        <div
          className={cn(
            'border-subtle pl-3',
            node.children.length === 0 ? 'ml-1.5' : 'ml-2 border-l'
          )}
        >
          <div className="flex flex-col gap-1">
            {node.children.map((child) => (
              <Row key={child.path} node={child} />
            ))}
            {node.children.length === 0 && (
              <div className="text-light text-sm">No data to show</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function makeFauxValueNode(node: SchemaNode): ValueNode {
  return { kind: 'value', name: node.name, path: node.path, type: 'object' };
}
