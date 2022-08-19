/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Link, Typography, Box } from '@material-ui/core'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import styles from '../styles.module.scss'

type Props = {
  retrievalMode: string
  setOpen: () => void
}

export const Header = (props: Props) => {
  return (
    <div className={styles.root}>
      <Box mb={3}>
        <Typography paragraph>
          Only select parameters can be changed here.
        </Typography>
        <Typography paragraph>
          However, you can still see the{' '}
          <Link href="#" underline="always" onClick={props.setOpen}>
            full configuration file{' '}
          </Link>{' '}
          (with sensitive values masked).
        </Typography>
        <Typography paragraph>
          <strong>Data retrieval mode</strong>: {props.retrievalMode}
        </Typography>
      </Box>
      <SectionTitle level={2} tag="h2" text="Section global" />
    </div>
  )
}
