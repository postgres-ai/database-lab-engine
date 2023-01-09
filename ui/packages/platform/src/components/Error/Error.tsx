/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import { Paper, Typography } from '@material-ui/core'

import { ClassesType } from '@postgres.ai/platform/src/components/types'
import { ErrorProps } from 'components/Error/ErrorWrapper'

interface ErrorWithStylesProps extends ErrorProps {
  classes: ClassesType
}

class Error extends Component<ErrorWithStylesProps> {
  render() {
    const { classes } = this.props

    return (
      <div>
        <Paper className={classes.paper}>
          <Typography
            variant="h1"
            component="div"
            className={classes.errorHead}
          >
            ERROR {this.props.code ? this.props.code : null}
          </Typography>

          <br />

          <Typography component="p" className={classes.errorText}>
            {this.props.message
              ? this.props.message
              : 'Unknown error occurred. Please try again later.'}
          </Typography>
        </Paper>
      </div>
    )
  }
}

export default Error
