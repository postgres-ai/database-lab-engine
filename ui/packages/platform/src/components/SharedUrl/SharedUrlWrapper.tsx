import { makeStyles } from '@material-ui/core'
import SharedUrl from 'components/SharedUrl/SharedUrl'
import { RouteComponentProps } from 'react-router'

interface MatchProps {
  url_uuid: string
}
export interface SharedUrlProps extends RouteComponentProps<MatchProps> {}

export const SharedUrlWrapper = (props: SharedUrlProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      container: {
        display: 'flex',
        flexWrap: 'wrap',
      },
      textField: {
        marginLeft: theme.spacing(1),
        marginRight: theme.spacing(1),
        width: '80%',
      },
      dense: {
        marginTop: 16,
      },
      menu: {
        width: 200,
      },
      updateButtonContainer: {
        marginTop: 20,
        textAlign: 'right',
      },
      errorMessage: {
        color: 'red',
      },
      orgsHeader: {
        position: 'relative',
      },
      newOrgBtn: {
        position: 'absolute',
        top: 0,
        right: 10,
      },
      banner: {
        height: 50,
        position: 'absolute',
        left: 0,
        bottom: 0,
        right: 0,
        backgroundColor: 'rgba(1, 58, 68, 0.8)',
        color: 'white',
        zIndex: 100,
        fontSize: 18,
        lineHeight: '50px',
        paddingLeft: 20,
        '& a': {
          color: 'white',
          fontWeight: '600',
        },
        '& svg': {
          position: 'absolute',
          right: 18,
          top: 18,
          cursor: 'pointer',
        },
      },
      signUpButton: {
        backgroundColor: 'white',
        fontWeight: 600,
        marginLeft: 10,
        '&:hover': {
          backgroundColor: '#ecf6f7',
        },
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <SharedUrl {...props} classes={classes} />
}
