import { makeStyles } from '@material-ui/core'
import { theme } from '@postgres.ai/shared/styles/theme'
import { colors } from '@postgres.ai/shared/styles/colors'
import { CSSProperties } from '@material-ui/styles'
import ProductCard from 'components/ProductCard/ProductCard'

export interface ProductCardProps {
  children?: React.ReactNode
  inline: boolean
  title: string
  icon?: JSX.Element | string
  style?: CSSProperties | undefined
  className?: string
  actions?: {
    id: string
    content: JSX.Element | string
  }[]
}

export const ProductCardWrapper = (props: ProductCardProps) => {
  const useStyles = makeStyles(
    (muiTheme) => ({
      root: {
        '& h1': {
          fontSize: '16px',
          margin: '0',
        },
        [muiTheme.breakpoints.down('xs')]: {
          height: '100%',
        },
        fontFamily: theme.typography.fontFamily,
        fontSize: '14px',
        border: '1px solid ' + colors.consoleStroke,
        maxWidth: '450px',
        minHeight: '260px',
        paddingTop: '15px',
        paddingBottom: '15px',
        paddingLeft: '20px',
        paddingRight: '20px',
        alignItems: 'center',
        borderRadius: '3px',
        // Flexbox.
        display: 'flex',
        '-webkit-flex-direction': 'column',
        '-ms-flex-direction': 'column',
        'flex-direction': 'column',
        '-webkit-flex-wrap': 'nowrap',
        '-ms-flex-wrap': 'nowrap',
        'flex-wrap': 'nowrap',
        '-webkit-justify-content': 'space-between',
        '-ms-flex-pack': 'justify',
        'justify-content': 'space-between',
        '-webkit-align-content': 'flex-start',
        '-ms-flex-line-pack': 'start',
        'align-content': 'flex-start',
        '-webkit-align-items': 'flex-start',
        '-ms-flex-align': 'start',
        'align-items': 'flex-start',
      },
      block: {
        marginBottom: '20px',
      },
      icon: {
        padding: '5px',
      },
      actionsContainer: {
        marginTop: '15px',

        [muiTheme.breakpoints.down('xs')]: {
          marginTop: '5px',
        },
      },
      contentContainer: {
        marginTop: '15px',
      },
      bottomContainer: {
        width: '100%',
        display: 'flex',
        '-webkit-flex-direction': 'row',
        '-ms-flex-direction': 'row',
        'flex-direction': 'row',
        '-webkit-flex-wrap': 'wrap',
        '-ms-flex-wrap': 'wrap',
        'flex-wrap': 'wrap',
        '-webkit-justify-content': 'space-between',
        '-ms-flex-pack': 'justify',
        'justify-content': 'space-between',
        '-webkit-align-content': 'stretch',
        '-ms-flex-line-pack': 'stretch',
        'align-content': 'stretch',
        '-webkit-align-items': 'flex-end',
        '-ms-flex-align': 'end',
        'align-items': 'flex-end',

        [muiTheme.breakpoints.down('xs')]: {
          flexDirection: 'column',
          alignItems: 'center',
        },
      },
      buttonSpan: {
        '&:not(:first-child)': {
          marginLeft: '15px',
        },

        [muiTheme.breakpoints.down('xs')]: {
          '&:not(:first-child)': {
            marginLeft: 0,
          },

          '& button': {
            marginTop: '15px',
            width: '100%',
          },
        },
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()

  return <ProductCard {...props} classes={classes} />
}
