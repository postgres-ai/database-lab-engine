/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'

import { Button } from '@postgres.ai/shared/components/Button2'
import { Modal } from '@postgres.ai/shared/components/Modal'

import { Content } from './Content'

type Props = {
  className?: string
}

export const ConnectModal = (props: Props) => {
  const [isOpen, setIsOpen] = useState(false)

  const handleClickOpen = () => setIsOpen(true)

  const handleClose = () => setIsOpen(false)

  return (
    <>
      <Button onClick={handleClickOpen} className={props.className}>
        Connect
      </Button>
      <Modal isOpen={isOpen} onClose={handleClose} title="Connection info">
        <Content />
      </Modal>
    </>
  )
}
