import React, { useState, useRef, useEffect } from 'react'
import { makeStyles } from '@material-ui/core'
import SendRoundedIcon from '@material-ui/icons/SendRounded';
import IconButton from "@material-ui/core/IconButton";
import { TextField } from '@postgres.ai/shared/components/TextField'
import {
  checkIsSendCmd,
  checkIsNewLineCmd,
  addNewLine,
  checkIsPrevMessageCmd,
  checkIsNextMessageCmd,
} from './utils'
import { useBuffer } from './useBuffer'
import { useCaret } from './useCaret'
import { theme } from "@postgres.ai/shared/styles/theme";
import { isMobileDevice } from "../../../utils/utils";


type Props = {
  sendDisabled: boolean
  onSend: (value: string) => void
  threadId?: string
}


const useStyles = makeStyles((theme) => (
  {
    root: {
      display: 'flex',
      alignItems: 'flex-end',
      marginTop: '20px',
      border: '1px solid rgba(0, 0, 0, 0.23)',
      borderRadius: '8px',
      '& .MuiOutlinedInput-root': {
        '& fieldset': {
          border: 'none',
        },
        '&:hover fieldset': {
          border: 'none',
        },
        '&.Mui-focused fieldset': {
          border: 'none',
        }
      }
    },
    field: {
      margin: '0 8px 0 0',
      flex: '1 1 100%',
      fontSize: '0.875rem',
    },
    fieldInput: {
      fontSize: '0.875rem',
      lineHeight: 'normal',
      height: 'auto',
      padding: '12px',
      [theme.breakpoints.down('sm')]: {
        fontSize: '1rem'
      }
    },
    iconButton: {
      height: 40,
      width: 40,
      fontSize: 24,
      transition: '.2s ease',
      '&:hover': {
        color: '#000'
      }
    },
    button: {
      flex: '0 0 auto',
      height: '40px',
    },
  })
)

export const Command = React.memo((props: Props) => {
  const { sendDisabled, onSend, threadId } = props

  const classes = useStyles()
  const isMobile = isMobileDevice();
  // Handle value.
  const [value, setValue] = useState('')

  // Input DOM Element reference.
  const inputRef = useRef<HTMLTextAreaElement | HTMLInputElement>()

  // Messages buffer.
  const buffer = useBuffer()

  // Input caret.
  const caret = useCaret(inputRef)

  const triggerSend = () => {
    if (!value.trim().length || sendDisabled) return

    onSend(value)
    buffer.addNew()
    setValue(buffer.getCurrent())
  }

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setValue(e.target.value)
    buffer.switchToLast()
    buffer.setToCurrent(e.target.value)
  }

  const handleBlur = () => {
    if ((window.innerWidth < theme.breakpoints.values.sm) && isMobile) {
      window.scrollTo({
        top: 0,
        behavior: 'smooth'
      })
      const footer: HTMLElement | null = document.querySelector("footer")
      if (footer) footer.style.display = 'flex';
    }
  }

  const handleFocus = () => {
    if ((window.innerWidth < theme.breakpoints.values.sm)  && isMobile) {
      const footer: HTMLElement | null = document.querySelector("footer")
      if (footer) footer.style.display = 'none';
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!inputRef.current) return

    // Trigger to send.
    if (checkIsSendCmd(e.nativeEvent)) {
      e.preventDefault()
      triggerSend()
      return
    }

    // Trigger line break.
    if (checkIsNewLineCmd(e.nativeEvent)) {
      e.preventDefault()

      const content = addNewLine(value, inputRef.current)

      setValue(content.value)
      caret.setPosition(content.caretPosition)
      return
    }

    // Trigger to use prev message.
    if (checkIsPrevMessageCmd(e.nativeEvent, inputRef.current)) {
      e.preventDefault()

      const prevValue = buffer.switchPrev()
      setValue(prevValue)
      return
    }

    // Trigger to use next message.
    if (checkIsNextMessageCmd(e.nativeEvent, inputRef.current)) {
      e.preventDefault()

      const nextValue = buffer.switchNext()
      setValue(nextValue)
      return
    }

    // Skip other keyboard events to fill input.
  }

  // Autofocus.
  useEffect(() => {
    if (!inputRef.current) return
    if (window.innerWidth > theme.breakpoints.values.md) inputRef.current.focus()
  }, [threadId]);


  return (
    <div className={classes.root}>
      <TextField
        autoFocus={window.innerWidth > theme.breakpoints.values.sm}
        multiline
        className={classes.field}
        onKeyDown={handleKeyDown}
        onChange={handleChange}
        onBlur={handleBlur}
        onFocus={handleFocus}
        InputProps={{
          inputRef,
          classes: {
            input: classes.fieldInput,
          },
        }}
        value={value}
        placeholder="Message..."
      />
      <IconButton
        onClick={triggerSend}
        className={classes.iconButton}
        disabled={sendDisabled || value.length === 0}
      >
        <SendRoundedIcon />
      </IconButton>
    </div>
  )
})
