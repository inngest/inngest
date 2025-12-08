import { cn } from '../../utils/classNames';
import { useExpansion } from '../ExpansionContext';
import type { ObjectNode, SchemaNode, ValueNode } from '../types';
import { CollapsibleRowWidget } from './CollapsibleRowWidget';
import { Row } from './Row';
import { ValueRow } from './ValueRow';

export type ObjectRowProps = { node: ObjectNode; typeLabelOverride?: string };

export function ObjectRow({ node, typeLabelOverride }: ObjectRowProps): React.ReactElement {
  const { isExpanded, toggleRecursive } = useExpansion();

  const open = isExpanded(node.path);

  const handleToggle = () => {
    const allDescendantPaths = collectAllExpandablePaths(node);
    toggleRecursive(node.path, allDescendantPaths);
  };

  return (
    <div className="flex flex-col gap-1">
      <div
        className="hover:bg-canvasSubtle flex cursor-pointer items-center rounded"
        onClick={handleToggle}
      >
        <CollapsibleRowWidget open={open} />
        <div className="-ml-0.5 flex-1">
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

function collectAllExpandablePaths(node: SchemaNode): string[] {
  const paths: string[] = [];

  function traverse(n: SchemaNode) {
    if (n.kind === 'object') {
      paths.push(n.path);
      n.children.forEach(traverse);
    } else if (n.kind === 'array') {
      paths.push(n.path);
      traverse(n.element);
    } else if (n.kind === 'tuple') {
      paths.push(n.path);
      n.elements.forEach(traverse);
    }
  }

  // Start with children to avoid adding the current node's path
  if (node.kind === 'object') {
    node.children.forEach(traverse);
  }

  return paths;
}
