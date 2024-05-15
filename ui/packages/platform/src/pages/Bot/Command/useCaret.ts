import { useState, useEffect, MutableRefObject } from 'react'

export const useCaret = (
  elementRef: MutableRefObject<HTMLInputElement | HTMLTextAreaElement | undefined>,
) => {
  // Keep caret position after making new line, but only after react update.
  const [nextPosition, setNextPosition] = useState<number | null>(null)

  useEffect(() => {
    if (nextPosition === null) return
    if (!elementRef.current) return

    elementRef.current.selectionStart = nextPosition
    elementRef.current.selectionEnd = nextPosition

    setNextPosition(null)
  }, [elementRef, nextPosition])

  return {
    setPosition: setNextPosition,
  }
}
