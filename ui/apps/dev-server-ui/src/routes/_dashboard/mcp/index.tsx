import { createFileRoute } from '@tanstack/react-router';
import MCPPage from '@/components/MCP/MCPPage';

export const Route = createFileRoute('/_dashboard/mcp/')({
  component: MCPComponent,
});

function MCPComponent() {
  return <MCPPage />;
}
