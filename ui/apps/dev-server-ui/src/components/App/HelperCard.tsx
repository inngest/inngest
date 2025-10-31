import { Card } from '@inngest/components/Card';
import { Link, LinkComponentProps } from '@tanstack/react-router';

type HelperCardProps = {
  title: string;
  description: string;
  icon: React.ReactNode;
  to?: LinkComponentProps['to'];
  href?: LinkComponentProps['href'];
  onClick?: () => void;
};

export default function HelperCard({
  title,
  description,
  icon,
  to,
  href,
  onClick,
}: HelperCardProps) {
  return (
    <Card className="hover:bg-canvasSubtle flex flex-col">
      <Link className="block p-4" to={to} href={href} onClick={onClick}>
        {icon}
        <p className="mb-1 mt-3">{title}</p>
        <p className="text-muted text-sm">{description}</p>
      </Link>
    </Card>
  );
}
