/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import classNames from 'classnames'
import { Link, Typography } from '@material-ui/core'
import Box from '@mui/material/Box'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { ExternalIcon } from '@postgres.ai/shared/icons/External'

import styles from '../styles.module.scss'

type Props = {
  retrievalMode: string
  setOpen: () => void
}

export const ConfigSectionTitle = ({ tag }: { tag: string }) => (
  <SectionTitle
    level={2}
    tag="h2"
    text={
      <div className={styles.sectionTitle}>
        <p>Section</p>
        <p>"{tag}"</p>
      </div>
    }
  />
)

const DOCS_URL =
  'https://postgres.ai/docs/reference-guides/database-lab-engine-configuration-reference'

export const Header = (props: Props) => (
  <div className={styles.root}>
    <Box mb={3}>
      <Typography paragraph>
        Only select parameters can be changed here.
      </Typography>
      <Typography paragraph>
        However, you can still see{' '}
        <Link
          href="#"
          underline="always"
          onClick={props.setOpen}
          className={styles.externalLink}
        >
          the full config
        </Link>
        . For details, read{' '}
        <a href={DOCS_URL} target="_blank" className={styles.externalLink}>
          the docs
          <ExternalIcon className={styles.externalIcon} />
        </a>
        .
      </Typography>
      <Typography paragraph>
        <strong>Data retrieval mode</strong>: {props.retrievalMode}
      </Typography>
    </Box>
    <ConfigSectionTitle tag="global" />
  </div>
)

export const ModalTitle = () => (
  <div>
    <Typography className={styles.modalTitle}>
      Full configuration file (view only)
    </Typography>
    <Typography variant="h3">
      Sensitive values are masked. For details, read{' '}
      <a href={DOCS_URL} target="_blank" className={styles.externalLink}>
        the docs
        <ExternalIcon
          className={classNames(styles.externalIcon, styles.largeIcon)}
        />
      </a>
      .
    </Typography>
  </div>
)
