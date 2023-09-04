/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles, Button } from '@material-ui/core'
import { useHistory } from 'react-router-dom'

import { colors } from '@postgres.ai/shared/styles/colors'
import { icons } from '@postgres.ai/shared/styles/icons'

import { ROUTES } from 'config/routes'

const useStyles = makeStyles(
  {
    demoNoticeText: {
      marginLeft: '0px',
      display: 'inline-block',
      position: 'relative',
      backgroundColor: colors.blue,
      color: colors.secondary2.darkDark,
      width: '100%',
      fontSize: '12px',
      lineHeight: '24px',
      fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
      paddingLeft: '10px',
      paddingTop: '4px',
      paddingBottom: '4px',
      '& > svg': {
        verticalAlign: 'baseline',
        marginBottom: '-1px',
        marginLeft: '0px',
        marginRight: '4px',
      },
    },
    demoOrgNoticeButton: {
      padding: '2px',
      paddingLeft: '6px',
      paddingRight: '6px',
      borderRadius: '3px',
      marginLeft: '5px',
      marginTop: '-2px',
      height: '20px',
      lineHeight: '20px',
      fontSize: '12px',
      fontWeight: 'bold',
    },
    noWrap: {
      whiteSpace: 'nowrap',
    },
  },
  { index: 1 },
)

export const DemoOrgNotice = () => {
  const classes = useStyles()
  const history = useHistory()

  const goToOrgForm = () => history.push(ROUTES.CREATE_ORG.path)

  return (
    <div className={classes.demoNoticeText}>
      {icons.infoIconBlue}&nbsp;This is a Demo organization, once you’ve
      explored <span className={classes.noWrap}>Database Lab</span> features:
      <Button
        variant="contained"
        color="primary"
        className={classes.demoOrgNoticeButton}
        onClick={goToOrgForm}
      >
        Create new organization
      </Button>
    </div>
  )
}
