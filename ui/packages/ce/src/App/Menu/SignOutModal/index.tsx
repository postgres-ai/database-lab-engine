import { Modal } from '@postgres.ai/shared/components/Modal'
import { Text } from '@postgres.ai/shared/components/Text'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'

export const SignOutModal = ({
  handleSignOut,
  onClose,
  isOpen,
}: {
  handleSignOut: () => void
  onClose: () => void
  isOpen: boolean
}) => {
  return (
    <Modal title={'Sign out'} onClose={onClose} isOpen={isOpen}>
      <Text>
        Are you sure you want to sign out? You will be redirected to the login
        page.
      </Text>
      <SimpleModalControls
        items={[
          {
            text: 'Cancel',
            onClick: onClose,
          },
          {
            text: 'Sign out',
            variant: 'primary',
            onClick: handleSignOut,
          },
        ]}
      />
    </Modal>
  )
}
