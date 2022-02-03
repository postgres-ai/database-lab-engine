/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import cn from 'classnames'

import styles from '@postgres.ai/shared/components/Link2/styles.module.scss'

type Props = React.DetailedHTMLProps<
  React.AnchorHTMLAttributes<HTMLAnchorElement>,
  HTMLAnchorElement
>

export const GatewayLink = (props: Props) => {
  const {
    rel = 'noopener noreferrer',
    target = '_blank',
    className,
    ...otherProps
  } = props

  return (
    <a
      {...otherProps}
      target={target}
      rel={rel}
      className={cn(styles.root, className)}
    />
  )
}
