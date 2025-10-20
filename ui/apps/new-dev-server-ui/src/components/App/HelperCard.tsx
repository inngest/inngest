import { Card } from '@inngest/components/Card'
import { Link } from '@tanstack/react-router'

type HelperCardProps = {
  title: string
  description: string
  icon: React.ReactNode
  href: string
  onClick?: () => void
}

export default function HelperCard({
  title,
  description,
  icon,
  href,
  onClick,
}: HelperCardProps) {
  return (
    <Card className="hover:bg-canvasSubtle flex flex-col">
      <Link className="block p-4" to={href} onClick={onClick}>
        {icon}
        <p className="mb-1 mt-3">{title}</p>
        <p className="text-muted text-sm">{description}</p>
      </Link>
    </Card>
  )
}
