import React, { useState, useRef, useEffect } from 'react'
import { makeStyles } from '@material-ui/core'

import { Button } from '@postgres.ai/shared/components/Button'
import { TextField } from '@postgres.ai/shared/components/TextField'

import { useFloatingIntercom } from 'hooks/useFloatingIntercom'

import {
  checkIsSendCmd,
  checkIsNewLineCmd,
  addNewLine,
  checkIsPrevMessageCmd,
  checkIsNextMessageCmd,
} from './utils'

import { useBuffer } from './useBuffer'
import { useCaret } from './useCaret'

type Props = {
  isDisabled: boolean
  onSend: (value: string) => void
}

const LABEL_FONT_SIZE = '14px'

const useStyles = makeStyles(
  {
    root: {
      display: 'flex',
      alignItems: 'flex-end',
      marginTop: '20px',
    },
    field: {
      margin: '0 8px 0 0',
      flex: '1 1 100%',
      fontSize: LABEL_FONT_SIZE,
    },
    fieldInput: {
      fontSize: '14px',
      lineHeight: 'normal',
      height: 'auto',
      padding: '12px',
    },
    fieldLabel: {
      fontSize: LABEL_FONT_SIZE,
    },
    button: {
      flex: '0 0 auto',
      height: '40px',
    },
  },
  { index: 1 },
)

export const Command = React.memo((props: Props) => {
  const { isDisabled, onSend } = props

  const classes = useStyles()

  // Handle value.
  const [value, setValue] = useState('')

  // Input DOM Element reference.
  const inputRef = useRef<HTMLTextAreaElement | HTMLInputElement>()

  // Messages buffer.
  const buffer = useBuffer()

  // Input caret.
  const caret = useCaret(inputRef)

  const triggerSend = () => {
    if (!value.trim().length) return

    onSend(value)
    buffer.addNew()
    setValue(buffer.getCurrent())
  }

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setValue(e.target.value)
    buffer.switchToLast()
    buffer.setToCurrent(e.target.value)
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

    inputRef.current.focus()
  }, [])

  // Floating intercom.
  const sendButtonRef = useRef<HTMLButtonElement | null>(null)

  useFloatingIntercom(sendButtonRef)

  return (
    <div className={classes.root}>
      <TextField
        autoFocus={true}
        label="Command"
        multiline
        className={classes.field}
        onKeyDown={handleKeyDown}
        disabled={isDisabled}
        InputProps={{
          inputRef,
          classes: {
            input: classes.fieldInput,
          },
        }}
        InputLabelProps={{
          className: classes.fieldLabel,
        }}
        value={value}
        onChange={handleChange}
      />
      <Button
        className={classes.button}
        onClick={triggerSend}
        ref={sendButtonRef}
      >
        Send
      </Button>
    </div>
  )
})
