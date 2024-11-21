/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Link as LinkBase, LinkProps } from 'react-router-dom'
import cn from 'classnames'

import styles from './styles.module.scss'

type Props = LinkProps & { external?: boolean }

export const Link = (props: Props) => {
  const { className, external, to, ...otherProps } = props

  if (external) {
    return (
      <a
        href={to as string}
        {...otherProps}
        className={cn(styles.root, className)}
        target="_blank"
        rel="noopener noreferrer"
      />
    )
  }

  return <LinkBase to={to} {...otherProps} className={cn(styles.root, className)} />
}
