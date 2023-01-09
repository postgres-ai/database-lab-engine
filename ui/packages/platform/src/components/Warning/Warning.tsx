/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'

import { icons } from '@postgres.ai/shared/styles/icons'
import { WarningProps } from 'components/Warning/WarningWrapper'
import { ClassesType } from '@postgres.ai/platform/src/components/types'

interface WarningWithStylesProps extends WarningProps {
  classes: ClassesType
}

class Warning extends Component<WarningWithStylesProps> {
  render() {
    const { classes, children, actions, inline } = this.props

    return (
      <div className={(!inline ? `${classes?.block} ` : '') + classes?.root}>
        {icons.warningIcon}

        <div className={classes?.container}>{children}</div>

        {actions ? (
          <span className={classes?.actions}>
            {actions.map((a, key) => {
              return (
                <span key={key} className={classes?.pageTitleActionContainer}>
                  {a}
                </span>
              )
            })}
          </span>
        ) : null}
      </div>
    )
  }
}
export default Warning
