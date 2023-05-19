import { makeStyles } from '@material-ui/core'

import { initialState } from '../reducer'

const useStyles = makeStyles((theme) => ({
  snippetContainer: {
    width: '100%',
    height: '100%',
    maxWidth: '800px',
    display: 'flex',
    flexDirection: 'row',
    justifyContent: 'space-between',
    gap: 40,

    [theme.breakpoints.down('sm')]: {
      flexDirection: 'column',
    },

    '&  p:first-child': {
      marginTop: '0',
    },
  },
  navigation: {
    display: 'flex',
    flexDirection: 'column',
    marginLeft: '-20px',
    flex: '0 0 220px',

    [theme.breakpoints.down('sm')]: {
      flex: 'auto',
    },

    '& span': {
      display: 'flex',
      alignItems: 'center',
      gap: 10,
      cursor: 'pointer',
      padding: '8px 14px 8px 20px',
      borderBottom: '1px solid #CCD7DA',
      transition: 'background-color 150ms cubic-bezier(0.4, 0, 0.2, 1) 0ms',

      '&:hover': {
        backgroundColor: '#F5F8FA',
      },
    },
  },
  form: {
    flex: '1 1 0',
    overflow: 'auto',

    [theme.breakpoints.down('sm')]: {
      flex: 'auto',
    },
  },
  active: {
    backgroundColor: '#F5F8FA',
    borderRight: '4px solid #FF6212',
  },
}))

export const DblabInstanceFormCreation = ({
  formStep,
  setFormStep,
  children,
}: {
  formStep: string
  setFormStep: (step: string) => void
  children: React.ReactNode
}) => {
  const classes = useStyles()

  return (
    <div className={classes.snippetContainer}>
      <div className={classes.navigation}>
        <span
          className={formStep === 'docker' ? classes.active : ''}
          onClick={() => setFormStep('docker')}
        >
          <img
            src={`/images/docker.svg`}
            width={30}
            height="auto"
            alt={'docker setup'}
          />
          Docker
        </span>
        <span
          className={formStep === 'ansible' ? classes.active : ''}
          onClick={() => setFormStep('ansible')}
        >
          <img
            src={`/images/ansible.svg`}
            width={30}
            height="auto"
            alt={'ansible setup'}
          />
          Ansible
        </span>
        {/* <span
          className={formStep === 'browser' ? classes.active : ''}
          onClick={() => setFormStep('browser')}
        >
          <img
            src={`/images/globe.svg`}
            width={30}
            height="auto"
            alt={'simple setup'}
          />
          Simple setup
        </span> */}
      </div>
      <div className={classes.form}>{children}</div>
    </div>
  )
}
