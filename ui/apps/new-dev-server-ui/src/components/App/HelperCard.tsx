import type { UrlObject } from 'url'
import NextLink from 'next/link'
import { Card } from '@inngest/components/Card'

type HelperCardProps = {
  title: string
  description: string
  icon: React.ReactNode
  href: string | UrlObject
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
      <NextLink className="block p-4" href={href} onClick={onClick}>
        {icon}
        <p className="mb-1 mt-3">{title}</p>
        <p className="text-muted text-sm">{description}</p>
      </NextLink>
    </Card>
  )
}
