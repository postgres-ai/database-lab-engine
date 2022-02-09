import { useRef } from 'react'

type Buffer = {
  values: string[]
  position: number
}

const NEW_EMPTY_VALUE = ''

const INITIAL_BUFFER: Buffer = {
  values: [NEW_EMPTY_VALUE],
  position: 0,
}

export const useBuffer = () => {
  const { current: buffer } = useRef<Buffer>(INITIAL_BUFFER)

  const getCurrent = () => buffer.values[buffer.position]

  const setToCurrent = (value: string) => buffer.values[buffer.position] = value

  const switchNext = () => {
    const newPosition = buffer.position + 1
    if (newPosition in buffer.values) buffer.position = newPosition
    return getCurrent()
  }

  const switchPrev = () => {
    const newPosition = buffer.position - 1
    if (newPosition in buffer.values) buffer.position = newPosition
    return getCurrent()
  }

  const switchToLast = () => {
    const lastIndex = buffer.values.length - 1
    buffer.position = lastIndex
    return getCurrent()
  }

  const addNew = () => {
    buffer.values.push(NEW_EMPTY_VALUE)
    return switchToLast()
  }

  return {
    switchNext,
    switchPrev,
    switchToLast,
    addNew,
    getCurrent,
    setToCurrent,
  }
}
