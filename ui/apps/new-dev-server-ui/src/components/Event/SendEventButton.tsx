import { useState } from 'react'
import { Button } from '@inngest/components/Button/NewButton'
import type { ButtonAppearance } from '@inngest/components/Button/NewButton'

import SendEventModal from '@/components/Event/SendEventModal'

type SendEventButtonProps = {
  data?: string | null
  label: string
  appearance?: ButtonAppearance
}

export default function SendEventButton({
  data,
  label,
  appearance = 'outlined',
}: SendEventButtonProps) {
  const [isSendEventModalVisible, setSendEventModalVisible] = useState(false)

  return (
    <>
      <Button
        label={label}
        kind="secondary"
        appearance={appearance}
        onClick={() => setSendEventModalVisible(true)}
      />
      {/* {isSendEventModalVisible && (
        <SendEventModal
          data={data}
          isOpen={isSendEventModalVisible}
          onClose={() => setSendEventModalVisible(false)}
        />
      )} */}
    </>
  )
}
