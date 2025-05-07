import { useState } from 'react'
import { makeStyles } from '@material-ui/core'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { Text } from '@postgres.ai/shared/components/Text'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'

type Props = {
  isOpen: boolean
  onClose: () => void
  instanceId: string
}

interface ErrorResponse {
  error?: {
    message?: string
    details?: string
  }
}

const useStyles = makeStyles(
  {
    errorMessage: {
      color: 'red',
      marginTop: '10px',
      wordBreak: 'break-all',
    },
    checkboxRoot: {
      padding: '9px 10px',
    },
    grayText: {
      color: '#8a8a8a',
      fontSize: '12px',
      wordBreak: 'break-word',
    },
    marginTop: {
      marginTop: '6px',
    },
  },
  { index: 1 },
)

export const ConfirmFullRefreshModal = ({
  isOpen,
  onClose,
  instanceId,
}: Props) => {
  const classes = useStyles()
  const stores = useStores()
  
  const { fullRefresh } = stores.main

  const [fullRefreshError, setFullRefreshError] = useState<ErrorResponse | null>(null)

  const handleClose = () => {
    onClose()
  }

  const handleConfirm = async () => {
    const result = await fullRefresh(instanceId);
    if (!result) {
      setFullRefreshError({ error: { message: 'Unexpected error occurred.' } });
      return;
    }
    const { response, error } = result;
    if (error) {
      setFullRefreshError({
        error: {
          message: error.message,
        },
      })
      return
    }
    if (response) {
      onClose()
    }
  }


  return (
    <Modal
      title={'Confirmation'}
      onClose={handleClose}
      isOpen={isOpen}
      size="sm"
    >
      <Text>
        Are you sure you want to perform a full refresh of the instance? 
        This action cannot be undone.
      </Text>
      {fullRefreshError && <p className={classes.errorMessage}>{fullRefreshError.error?.message}</p>}
      <SimpleModalControls
        items={[
          {
            text: 'Cancel',
            onClick: handleClose,
          },
          {
            text: 'Confirm',
            variant: 'primary',
            onClick: handleConfirm,
            isDisabled: fullRefreshError !== null,
          },
        ]}
      />
    </Modal>
  )
}
