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
      display: 'inline-block',
      maxWidth: '100%',
      width: '100%',

      '& code': {
        whiteSpace: 'inherit !important',
      },
    },
    copyButton: {
      position: 'absolute',
      top: 15,
      right: 4,
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

export const SyntaxHighlight = ({ content }: { content: string }) => {
  const classes = useStyles()

  const copyContent = () => {
    copyToClipboard(content.replace(/^\s*[\r\n]/gm, ''))
  }

  return (
    <div className={classes.copyFieldContainer}>
      <SyntaxHighlighter
        language="bash"
        wrapLines
        style={oneLight}
        customStyle={{ borderRadius: 4, margin: '12px 0', fontSize: '14px' }}
        codeTagProps={{
          style: {
            fontSize: '14px',
          },
        }}
        lineProps={{
          style: {
            display: 'flex',
            fontSize: '14px',
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
