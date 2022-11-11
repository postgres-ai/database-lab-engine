/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import { Tooltip, Button } from '@material-ui/core'

import { ClassesType } from '@postgres.ai/platform/src/components/types'
import { ConsoleButtonProps } from 'components/ConsoleButton/ConsoleButtonWrapper'

interface ConsoleButtonWithStylesProps extends ConsoleButtonProps {
  classes: ClassesType
}

class ConsoleButton extends Component<ConsoleButtonWithStylesProps> {
  render() {
    const { classes, title, children, ...other } = this.props

    // We have to use external tooltip component as disable button cannot show tooltip.
    // Details: https://material-ui.com/components/tooltips/#disabled-elements.
    return (
      <Tooltip classes={{ tooltip: classes.tooltip }} title={title}>
        <span>
          <Button {...other}>{children}</Button>
        </span>
      </Tooltip>
    )
  }
}

export default ConsoleButton
