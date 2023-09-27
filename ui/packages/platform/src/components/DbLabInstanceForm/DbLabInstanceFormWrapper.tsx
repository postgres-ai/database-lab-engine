/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { RouteComponentProps } from 'react-router'
import { makeStyles } from '@material-ui/core'

import DbLabInstanceForm from 'components/DbLabInstanceForm/DbLabInstanceForm'

import { styles } from '@postgres.ai/shared/styles/styles'

export interface DbLabInstanceFormProps {
  userID?: number
  edit?: boolean
  orgId: number
  project: string | undefined
  history: RouteComponentProps['history']
  orgPermissions: {
    dblabInstanceCreate?: boolean
  }
}

export const DbLabInstanceFormWrapper = (props: DbLabInstanceFormProps) => {
  const useStyles = makeStyles(
    {
      textField: {
        ...styles.inputField,
        maxWidth: 400,
      },
      errorMessage: {
        color: 'red',
      },
      fieldBlock: {
        width: '100%',
      },
      urlOkIcon: {
        marginBottom: -5,
        marginLeft: 10,
        color: 'green',
      },
      urlOk: {
        color: 'green',
      },
      urlFailIcon: {
        marginBottom: -5,
        marginLeft: 10,
        color: 'red',
      },
      urlFail: {
        color: 'red',
      },
      warning: {
        color: '#801200',
        fontSize: '0.9em',
      },
      warningIcon: {
        color: '#801200',
        fontSize: '1.2em',
        position: 'relative',
        marginBottom: -3,
      },
      container: {
        display: 'flex',
        flexDirection: 'row',
        justifyContent: 'space-between',
        marginTop: 30,
        gap: 60,
        width: '100%',
        height: '95%',
        position: 'relative',
        '& input': {
          padding: '13.5px 14px',
        },

        '@media (max-width: 1200px)': {
          flexDirection: 'column',
          height: 'auto',
          gap: 30,
        },
      },
      form: {
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        flex: '3 1 0',

        '& > [role="tabpanel"] .MuiBox-root': {
          padding: 0,

          '& > div:first-child': {
            marginTop: '10px',
          },
        },
      },
      activeBorder: {
        border: '1px solid #FF6212 !important',
      },
      providerFlex: {
        display: 'flex',
        gap: '10px',
        marginBottom: '20px',
        overflow: 'auto',
        flexShrink: 0,

        '& > div': {
          width: '100%',
          height: '100%',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          border: '1px solid #e0e0e0',
          padding: '15px',
          borderRadius: '4px',
          cursor: 'pointer',
          transition: 'border 0.3s ease-in-out',

          '@media (max-width: 600px)': {
            width: '122.5px',
            height: '96px',
          },

          '&:hover': {
            border: '1px solid #FF6212',
          },

          '& > img': {
            margin: 'auto',
          },
        },
      },
      sectionTitle: {
        fontSize: '14px',
        fontWeight: 500,
        marginTop: '20px',

        '&:first-child': {
          marginTop: 0,
        },
      },
      sectionContainer: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',

        '& > .MuiTabs-root, .MuiTabs-fixed': {
          overflow: 'auto !important',
        },

        '& span': {
          top: '40px',
          height: '2px',
        },
      },
      tab: {
        minWidth: 'auto',
        padding: '0 12px',
      },
      tabPanel: {
        padding: '10px 0 0 0',
      },
      instanceSize: {
        marginBottom: '10px',
        border: '1px solid #e0e0e0',
        borderRadius: '4px',
        cursor: 'pointer',
        padding: '15px',
        transition: 'border 0.3s ease-in-out',
        display: 'flex',
        gap: 10,
        flexDirection: 'column',

        '&:hover': {
          border: '1px solid #FF6212',
        },

        '& > p': {
          margin: 0,
        },

        '& > div': {
          display: 'flex',
          gap: 10,
          alignItems: 'center',
          flexWrap: 'wrap',
        },
      },
      serviceLocation: {
        display: 'flex',
        flexDirection: 'column',
        gap: '5px',
        marginBottom: '10px',
        border: '1px solid #e0e0e0',
        borderRadius: '4px',
        cursor: 'pointer',
        padding: '15px',
        transition: 'border 0.3s ease-in-out',

        '&:hover': {
          border: '1px solid #FF6212',
        },

        '& > p': {
          margin: 0,
        },
      },
      instanceParagraph: {
        margin: '0 0 10px 0',
      },
      filterSelect: {
        flex: '2 1 0',

        '& .MuiSelect-select': {
          padding: '10px',
        },

        '& .MuiInputBase-input': {
          padding: '10px',
        },

        '& .MuiSelect-icon': {
          top: 'calc(50% - 9px)',
        },
      },
      generateContainer: {
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
        gap: '10px',

        '& > button': {
          width: 'max-content',
          marginTop: '10px',
          flexShrink: 0,
          height: 'calc(100% - 10px)',
        },

        '@media (max-width: 640px)': {
          flexDirection: 'column',
          alignItems: 'flex-start',
          gap: 0,

          '& > button': {
            height: 'auto',
          },
        },
      },
      backgroundOverlay: {
        '&::before': {
          content: '""',
          position: 'absolute',
          top: 0,
          left: 0,
          width: '100%',
          height: '100%',
          background: 'rgba(255, 255, 255, 0.8)',
          zIndex: 1,
        },
      },
      absoluteSpinner: {
        position: 'fixed',
        top: '50%',
        left: '50%',
        transform: 'translate(-50%, -50%)',
        zIndex: 1,
        width: '32px !important',
        height: '32px !important',
      },
      marginTop: {
        marginTop: '10px',
      },
      sliderContainer: {
        width: '100%',
        padding: '30px 35px',
        borderRadius: '4px',
        border: '1px solid #e0e0e0',
      },
      sliderInputContainer: {
        display: 'flex',
        flexDirection: 'column',
        marginBottom: '20px',
        gap: '20px',
        maxWidth: '350px',
        width: '100%',
      },
      sliderVolume: {
        display: 'flex',
        flexDirection: 'row',
        gap: '10px',
        alignItems: 'center',
      },
      databaseSize: {
        display: 'flex',
        flexDirection: 'row',
        gap: '10px',
        alignItems: 'center',
        spinner: {
          marginLeft: 8,
          color: 'inherit',
        },
      },
    },
    { index: 1 },
  )

  const classes = useStyles()

  return <DbLabInstanceForm {...props} classes={classes} />
}
