/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/core'

import { Button } from '@postgres.ai/shared/components/Button2'

type Props = {
  isReloading: boolean
  onReload: () => void
}

const useStyles = makeStyles(
  {
    spinner: {
      margin: '5px',
    },
    content: {
      flex: '0 0 auto',
      alignSelf: 'flex-start',
    },
  },
  { index: 1 },
)

export const ModalReloadButton = (props: Props) => {
  const classes = useStyles()

  return (
    <Button
      onClick={props.onReload}
      className={classes.content}
      isLoading={props.isReloading}
      isDisabled={props.isReloading}
    >
      Reload info
    </Button>
  )
}
