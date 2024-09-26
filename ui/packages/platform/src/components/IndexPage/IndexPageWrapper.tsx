import { makeStyles } from '@material-ui/core'
import { colors } from '@postgres.ai/shared/styles/colors'
import IndexPage from 'components/IndexPage/IndexPage'
import { MatchParams, ProjectWrapperProps } from 'components/types'
import { RouteComponentProps } from 'react-router'

export interface IndexPageProps
  extends RouteComponentProps<MatchParams>,
    Omit<ProjectWrapperProps, 'location' | 'match'> {}

export const IndexPageWrapper = (props: IndexPageProps) => {
  const drawerWidth = 185

  const useStyles = makeStyles(
    (theme) => ({
      root: {
        flex: '1 1 0',
        zIndex: 1,
        overflow: 'hidden',
        position: 'relative',
        display: 'flex',
        fontSize: '14px',
      },
      appBar: {
        zIndex: theme.zIndex.drawer + 1,
        backgroundColor: colors.secondary2.darkDark,
      },
      drawerPaper: {
        [theme.breakpoints.up('md')]: {
          paddingTop: '40px',
        },
        position: 'absolute',
        overflow: 'hidden',
        width: drawerWidth,
        'background-color': colors.consoleMenuBackground,
        'border-right-color': colors.consoleStroke,
        '& hr': {
          backgroundColor: colors.consoleStroke,
        },
      },
      drawer: {
        minWidth: drawerWidth,
        flexShrink: 0,
      },
      drawerContainer: {
        minWidth: drawerWidth,
      },
      navIconHide: {
        [theme.breakpoints.down('sm')]: {
          display: 'inline-flex',
        },
        [theme.breakpoints.up('md')]: {
          display: 'none',
        },
        [theme.breakpoints.up('lg')]: {
          display: 'none',
        },
        marginLeft: '-14px',
        '& svg': {
          marginTop: '-4px',
        },
      },
      rightDivider: {
        marginLeft: 30,
      },
      navIconSignOut: {
        position: 'absolute',
        right: 0,
        padding: 8,
      },
      navIconArea: {
        position: 'absolute',
        right: 35,
        color: 'white',
        textDecoration: 'none',
      },
      navIconProfile: {
        padding: 8,
      },
      toolbar: theme.mixins.toolbar,
      topToolbar: {
        minHeight: 40,
        height: 40,
        paddingLeft: 14,
        paddingRight: 14,
        color: '#fff',
      },
      logo: {
        color: 'white',
        textDecoration: 'none',
        fontSize: 16,
      },
      userName: {
        position: 'absolute',
        right: 77,
        fontSize: 14,
        [theme.breakpoints.down('sm')]: {
          display: 'none',
        },
        [theme.breakpoints.up('md')]: {
          display: 'block',
        },
        [theme.breakpoints.up('lg')]: {
          display: 'block',
        },
      },
      orgHeaderContainer: {
        position: 'relative',
        height: 40,
      },
      orgHeader: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        position: 'absolute',
        left: '15px',
        top: '15px',
        fontStyle: 'normal',
        fontWeight: 'normal',
        fontSize: '10px',
        lineHeight: '12px',
        color: '#000000',
      },
      orgSwitcher: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        position: 'absolute',
        right: '10px',
        top: '10px',
        border: '1px solid #CCD7DA',
        borderRadius: '3px',
        fontStyle: 'normal',
        fontWeight: 'normal',
        fontSize: '10px',
        lineHeight: '12px',
        display: 'flex',
        alignItems: 'center',
        textAlign: 'center',
        color: '#808080',
        padding: 3,
        textTransform: 'capitalize',
        cursor: 'pointer',
      },
      orgNameContainer: {
        paddingLeft: '15px',
        height: '35px',
        position: 'relative',
      },
      orgName: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        fontStyle: 'normal',
        fontWeight: 'bold',
        fontSize: '14px',
        lineHeight: '16px',
        color: '#000000',
        maxWidth: '125px',
        whiteSpace: 'nowrap',
        textOverflow: 'ellipsis',
        overflow: 'hidden',
        display: 'inline-block',
      },
      orgPlan: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        fontStyle: 'normal',
        fontWeight: 'normal',
        fontSize: '10px',
        lineHeight: '11px',
        alignItems: 'center',
        textAlign: 'center',
        color: '#FFFFFF',
        backgroundColor: colors.secondary2.main,
        padding: '1px',
        borderRadius: '4px',
        paddingLeft: '3px',
        paddingRight: '3px',
        marginLeft: 10,
        position: 'absolute',
        top: '2px',
      },
      menuItemLabel: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        fontStyle: 'normal',
        fontWeight: 'normal',
        fontSize: '10px',
        lineHeight: '11px',
        alignItems: 'center',
        textAlign: 'center',
        color: '#FFFFFF',
        backgroundColor: colors.pgaiOrange,
        padding: '1px',
        borderRadius: '4px',
        paddingLeft: '3px',
        paddingRight: '3px',
        marginLeft: 10,
        position: 'absolute',
        top: '10px',
      },
      headerLinkMenuItemLabel: {
        position: 'static'
      },
      menuSectionHeader: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        fontStyle: 'normal',
        fontWeight: 'bold',
        fontSize: '14px',
        lineHeight: '16px',
        color: '#000000',
        padding: '0px',
        marginTop: '10px',
      },
      bottomFixedMenuItem: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        fontStyle: 'normal',
        fontWeight: 'bold',
        fontSize: '14px',
        lineHeight: '16px',
        color: '#000000',
        padding: '0px',
        marginTop: '0px',
      },
      menuSectionHeaderLink: {
        textDecoration: 'none',
        paddingTop: 12,
        paddingBottom: 12,
        paddingRight: 14,
        width: '100%',
        paddingLeft: '15px',
        color: '#000000',
        display: 'inline-flex',
        alignItems: 'center',
        whiteSpace: 'nowrap'
      },
      menuSectionHeaderActiveLink: {
        textDecoration: 'none',
        paddingTop: 12,
        paddingBottom: 12,
        paddingRight: 14,
        width: '100%',
        paddingLeft: '15px',
        color: '#000000',
      },
      menuSingleSectionHeaderActiveLink: {
        backgroundColor: colors.consoleStroke,
      },
      menuPointer: {
        height: '100%',
      },
      navMenu: {
        padding: '0px',
        height: 'calc(100% - 160px)',
        overflowY: 'auto',
        
        display: 'flex',
        flexDirection: 'column',
      },
      menuSectionHeaderIcon: {
        marginRight: '13px',
      },
      menuItem: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
        fontStyle: 'normal',
        fontWeight: 'normal',
        fontSize: '14px',
        lineHeight: '16px',
        color: '#000000',
        padding: '0px',
        position: 'relative',
      },
      menuItemLink: {
        textDecoration: 'none',
        paddingTop: 8,
        paddingBottom: 8,
        paddingRight: 14,
        width: '100%',
        paddingLeft: '43px',
        color: '#000000',
        position: 'relative',
      },
      menuItemActiveLink: {
        textDecoration: 'none',
        paddingTop: 8,
        paddingBottom: 8,
        paddingRight: 14,
        backgroundColor: colors.consoleStroke,
        width: '100%',
        paddingLeft: '43px',
        color: '#000000',
        position: 'relative',
      },
      betaContainer: {
        '& > svg': {
          display: 'block',
          margin: 'auto',
        },
        'font-family': '"Roboto", "Helvetica", "Arial", sans-serif',
        'font-style': 'normal',
        'font-weight': 'normal',
        'font-size': '16px',
        'max-width': '500px',
        background: '#ffffff',
        border: '1px solid ' + colors.consoleStroke,
        margin: 'auto',
        'border-radius': '3px',
        padding: '40px',
      },
      betaWrapper: {
        background: colors.consoleMenuBackground,
        height: '100vh',
        display: 'flex',
        'align-items': 'center',
      },
      tosContainer: {
        '& > svg': {
          display: 'block',
          margin: 'auto',
        },
        'font-family': '"Roboto", "Helvetica", "Arial", sans-serif',
        'font-style': 'normal',
        'font-weight': 'normal',
        'font-size': '14px',
        width: '330px',
        background: '#ffffff',
        border: '1px solid ' + colors.consoleStroke,
        margin: 'auto',
        'border-radius': '3px',
        paddingTop: '35px',
        paddingBottom: '35px',
        textAlign: 'center',
        '& > p': {
          marginTop: '0px',
        },
      },
      tosWrapper: {
        background: colors.consoleMenuBackground,
        height: '100vh',
        display: 'flex',
        'align-items': 'center',
        '& a': {
          textDecoration: 'underline',
        },
      },
      tosAgree: {
        marginTop: '15px',
        display: 'inline-block',
      },
      navBottomFixedMenu: {
        width: '100%',
        borderTop: '1px solid',
        borderColor: colors.consoleStroke,
        padding: 0,
        position: 'absolute',
        bottom: 0,
        backgroundColor: colors.consoleMenuBackground,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <IndexPage {...props} classes={classes} />
}