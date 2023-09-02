import { makeStyles } from '@material-ui/core'
import { colors } from '@postgres.ai/shared/styles/colors'
import { RouteComponentProps } from 'react-router'
import Dashboard from 'components/Dashboard/Dashboard'

export interface DashboardProps {
  org?: string | number
  orgId?: number
  onlyProjects?: boolean
  history: RouteComponentProps['history']
  project?: string | undefined
  orgPermissions?: {
    dblabInstanceCreate?: boolean
    checkupReportConfigure?: boolean
  }
}

export const DashboardWrapper = (props: DashboardProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      orgsHeader: {
        position: 'relative',
      },
      newOrgBtn: {
        position: 'absolute',
        top: 0,
        right: 10,
      },
      nameColumn: {
        'word-wrap': 'break-word',
        [theme.breakpoints.down('sm')]: {
          maxWidth: 'calc(100vw - 150px)',
        },
        [theme.breakpoints.up('md')]: {
          maxWidth: 'calc(100vw - 350px)',
        },
        [theme.breakpoints.up('lg')]: {
          maxWidth: 'calc(100vw - 350px)',
        },
        '& > a': {
          color: 'black',
          textDecoration: 'none',
        },
        '& > a:hover': {
          color: 'black',
          textDecoration: 'none',
        },
      },
      cell: {
        '& > a': {
          color: 'black',
          textDecoration: 'none',
        },
        '& > a:hover': {
          color: 'black',
          textDecoration: 'none',
        },
      },
      activityButton: {
        '&:not(:first-child)': {
          marginLeft: '15px',
        },
      },
      onboardingCard: {
        border: '1px solid ' + colors.consoleStroke,
        borderRadius: 3,
        padding: 15,
        '& h1': {
          fontSize: '16px',
          margin: '0',
        },
      },
      onboarding: {
        '& ul': {
          paddingInlineStart: '20px',
        },
      },
      filterOrgsInput: {
        width: '100%',

        '& .MuiOutlinedInput-input': {
          width: '200px',
        },
      },
      blockedStatus: {
        color: colors.state.error,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      validStatus: {
        color: colors.state.ok,
        fontSize: '1.1em',
        verticalAlign: 'middle',
        '& svg': {
          marginTop: '-3px',
        },
      },
      flexContainer: {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: '100%',
        height: '100%',
        gap: 40,
        marginTop: '20px',

        '& > div': {
          maxWidth: '300px',
          width: '100%',
          height: '100%',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          border: '1px solid #e0e0e0',
          padding: '20px',
          borderRadius: '4px',
          cursor: 'pointer',
          fontSize: '15px',
          transition: 'border 0.3s ease-in-out',

          '&:hover': {
            border: '1px solid #FF6212',
          },
        },
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <Dashboard {...props} classes={classes} />
}
