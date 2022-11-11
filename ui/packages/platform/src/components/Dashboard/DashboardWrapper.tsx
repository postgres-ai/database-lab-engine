import { makeStyles } from '@material-ui/core'
import { colors } from '@postgres.ai/shared/styles/colors'
import { RouteComponentProps } from 'react-router'
import Dashboard from 'components/Dashboard/Dashboard'

export interface DashboardProps {
  org: string | number
  orgId: number
  onlyProjects: boolean
  history: RouteComponentProps['history']
  project?: string | undefined
  orgPermissions: {
    dblabInstanceCreate?: boolean
    checkupReportConfigure?: boolean
  }
}

export const DashboardWrapper = (props: DashboardProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      stubContainerProjects: {
        marginRight: '-20px',
        paddingBottom: 0,
        [theme.breakpoints.down('sm')]: {
          flexDirection: 'column',
          marginRight: 0,
          marginTop: '-20px',
        },
      },
      productCardProjects: {
        flex: '1 1 100%',
        marginRight: '20px',
        [theme.breakpoints.down('sm')]: {
          flex: '0 0 auto',
          marginRight: 0,
          marginTop: '20px',
        },
      },
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
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <Dashboard {...props} classes={classes} />
}
