import { makeStyles } from '@material-ui/core'
import { styles } from '@postgres.ai/shared/styles/styles'
import { RouteComponentProps } from 'react-router'
import ReportFile from 'components/ReportFile/ReportFile'

interface MatchParams {
  fileId: string
  fileType: string
  reportId: string
}

export interface ReportFileProps extends RouteComponentProps<MatchParams> {
  projectId: string | number | undefined
  reportId: string
  fileType: string
  raw: boolean
}

export const ReportFileWrapper = (props: ReportFileProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        width: '100%',
        [theme.breakpoints.down('sm')]: {
          maxWidth: '100vw',
        },
        [theme.breakpoints.up('md')]: {
          maxWidth: 'calc(100vw - 240px)',
        },
        [theme.breakpoints.up('lg')]: {
          maxWidth: 'calc(100vw - 240px)',
        },
        minHeight: '100%',
        zIndex: 1,
        position: 'relative',
      },
      reportFileContent: {
        border: '1px solid silver',
        margin: 5,
        padding: 5,
      },
      code: {
        width: '100%',
        'background-color': 'rgb(246, 248, 250)',
        '& > div > textarea': {
          fontFamily:
            '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
            ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
          color: 'black',
          fontSize: 14,
        },
      },
      rawCode: {
        width: '100%',
        'background-color': 'none',
        'margin-top': '10px',
        padding: '0px',
        '& > .MuiOutlinedInput-multiline': {
          padding: '0px!important',
        },
        '& > div > textarea': {
          fontFamily:
            '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
            ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
          color: 'black',
          fontSize: 14,
        },
        '& > .MuiInputBase-fullWidth > fieldset': {
          borderWidth: 'none!important',
          borderStyle: 'none!important',
          borderRadius: '0px!important',
        },
      },
      bottomSpace: {
        ...styles.bottomSpace,
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <ReportFile {...props} classes={classes} />
}
