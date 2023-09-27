import copyToClipboard from 'copy-to-clipboard'
import { makeStyles, IconButton } from '@material-ui/core'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'

import { icons } from '@postgres.ai/shared/styles/icons'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'

const useStyles = makeStyles(
  {
    copyFieldContainer: {
      position: 'relative',
      display: 'inline-flex',
      maxWidth: '100%',
      width: '100%',
      margin: "12px 0",
      backgroundColor: 'rgb(250, 250, 250)',

      '& code': {
        whiteSpace: 'inherit !important',
      },
    },
    copyButton: {
      top: 6,
      position: 'sticky',
      right: 6,
      zIndex: 10,
      width: 26,
      height: 26,
      padding: 8,
      backgroundColor: 'rgba(128, 128, 128, 0.15)',
      transition: 'background-color 0.2s ease-in-out, color 0.2s ease-in-out',

      '&:hover': {
        backgroundColor: 'rgba(128, 128, 128, 0.25)',
      },
    },
  },
  { index: 1 },
)

export const SyntaxHighlight = ({
  content,
  wrapLines,
  id,
  style,
}: {
  content: string
  wrapLines?: boolean
  id?: string
  style?: React.CSSProperties
}) => {
  const classes = useStyles()

  const copyContent = () => {
    if (!content) {
      const codeTag = document.getElementById(id as string)
      if (codeTag) {
        copyToClipboard(codeTag.innerText)
      }
      return
    }

    copyToClipboard(content.replace(/^\s*[\r\n]/gm, ''))
  }

  const fontSize = style ? '12px' : '14px'

  return (
    <div className={classes.copyFieldContainer} style={style}>
      <SyntaxHighlighter
        id={id}
        language="bash"
        wrapLines={wrapLines}
        style={oneLight}
        customStyle={{
          borderRadius: 4,
          fontSize: fontSize,
          margin: 0,
          border: 0,
          flexGrow: 1,
          height: '100%',
        }}
        codeTagProps={{
          style: {
            fontSize: fontSize,
          },
        }}
        lineProps={{
          style: {
            display: 'flex',
            fontSize: fontSize,
          },
        }}
      >
        {content}
      </SyntaxHighlighter>
      <IconButton
        className={classes.copyButton}
        aria-label="Copy"
        onClick={copyContent}
      >
        <Tooltip content={'Copy'}>{icons.copyIcon}</Tooltip>
      </IconButton>
    </div>
  )
}
