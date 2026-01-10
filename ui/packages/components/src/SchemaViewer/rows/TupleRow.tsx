import { useExpansion } from '../ExpansionContext';
import type { SchemaNode, TupleNode, ValueNode } from '../types';
import { CollapsibleRowWidget } from './CollapsibleRowWidget';
import { Row } from './Row';
import { ValueRow } from './ValueRow';

export type TupleRowProps = { node: TupleNode; typeLabelOverride?: string };

export function TupleRow({ node, typeLabelOverride }: TupleRowProps): React.ReactElement {
  const { isExpanded, toggleRecursive } = useExpansion();
  const open = isExpanded(node.path);
  const hasChildren = node.elements.length > 0;

  const handleToggle = () => {
    if (!hasChildren) return;
    const allDescendantPaths = collectAllExpandablePaths(node);
    toggleRecursive(node.path, allDescendantPaths);
  };

  return (
    <div className="flex flex-col gap-1">
      <div
        className={`flex items-center rounded ${
          hasChildren ? 'hover:bg-canvasSubtle cursor-pointer' : ''
        }`}
        onClick={handleToggle}
      >
        {hasChildren ? <CollapsibleRowWidget open={open} /> : null}
        <div className="-ml-0.5 flex-1">
          <ValueRow
            boldName={open}
            node={makeFauxValueNode(node)}
            typeLabelOverride={typeLabelOverride ?? ''}
            typePillOverride={hasChildren ? `${node.elements.length} item tuple` : 'Tuple'}
          />
        </div>
      </div>
      {open && (
        <div className="border-subtle ml-2 border-l pl-3">
          <div className="flex flex-col gap-1">
            {node.elements.map((child) => (
              <Row key={child.path} node={child} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function makeFauxValueNode(node: TupleNode): ValueNode {
  return { kind: 'value', name: node.name, path: node.path, type: 'array' };
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

  // Start with elements to avoid adding the current node's path
  if (node.kind === 'tuple') {
    node.elements.forEach(traverse);
  }

  return paths;
}
