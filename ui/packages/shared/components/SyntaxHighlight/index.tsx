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

export const SyntaxHighlight = ({
  content,
  wrapLines,
}: {
  content: string
  wrapLines?: boolean
}) => {
  const classes = useStyles()

  return (
    <div className={classes.copyFieldContainer}>
      <SyntaxHighlighter
        language="javascript"
        wrapLines={wrapLines}
        style={oneLight}
        customStyle={{ borderRadius: 4, margin: '12px 0' }}
        lineProps={{
          style: {
            display: 'flex',
          },
        }}
      >
        {content}
      </SyntaxHighlighter>
      <IconButton
        className={classes.copyButton}
        aria-label="Copy"
        onClick={() => copyToClipboard(content)}
      >
        <Tooltip content={'Copy'}>{icons.copyIcon}</Tooltip>
      </IconButton>
    </div>
  )
}
