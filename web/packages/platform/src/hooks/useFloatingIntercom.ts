import { MutableRefObject, useRef } from 'react'
import { useInterval } from 'use-interval'
import { makeStyles } from '@material-ui/core'

import { createTransitionInteractive } from '@postgres.ai/shared/styles/vars'

const useStyles = makeStyles({
  intercom: {
    transition: createTransitionInteractive('transform'),
  },
})

const UPDATE_INTERVAL = 1000

// Intercom has two elements as launcher buttons.
const INTERCOM_BUTTON_SELECTOR = `div[aria-label="Open Intercom Messenger"]`
const INTERCOM_IFRAME_SELECTOR = `iframe[name="intercom-launcher-frame"]`

// Min indent between target and intercom launcher.
const MIN_INTERCOM_INDENT_PX = 8

// Check is intercom launcher intersected with target including "MIN_INTERCOM_INDENT_PX".
// Intercom position can be corrected by current shift "intercomShiftY".
const checkIsIntersectedY = (
  targetRect: DOMRect,
  intercomRect: DOMRect,
  intercomShiftY: number | undefined = 0,
) => {
  const targetTop = targetRect.top - MIN_INTERCOM_INDENT_PX
  const targetBottom = targetRect.bottom + MIN_INTERCOM_INDENT_PX

  const intercomTop = intercomRect.top + intercomShiftY
  const intercomBottom = intercomRect.bottom + intercomShiftY

  return !(targetBottom < intercomTop || intercomBottom < targetTop)
}

// Needed shift delta between target and intercom launcher based on the current position.
const getIntercomShiftYDelta = (targetRect: DOMRect, intercomRect: DOMRect) =>
  intercomRect.bottom - targetRect.top + MIN_INTERCOM_INDENT_PX

export const useFloatingIntercom = (
  targetRef: MutableRefObject<Element | null>,
) => {
  const classes = useStyles()

  const currentShiftYRef = useRef(0)

  // Recalculate intercom position every UPDATE_INTERVAL.
  useInterval(() => {
    if (!targetRef.current) return
    const targetRect = targetRef.current.getBoundingClientRect()

    // TODO?(Anton): Possible optimization search later.
    // Find intercom launcher element.
    const intercomElement = (window.document.querySelector(
      INTERCOM_BUTTON_SELECTOR,
    ) ?? window.document.querySelector(INTERCOM_IFRAME_SELECTOR)) as
      | HTMLDivElement
      | HTMLIFrameElement

    if (!intercomElement) return

    // Update intercom shift and keep it.
    const setShiftY = (shiftY: number) => {
      intercomElement.style.transform = `translateY(-${shiftY}px)`
      currentShiftYRef.current = shiftY
    }

    // Make intercom moving smooth.
    intercomElement.classList.add(classes.intercom)

    const intercomRect = intercomElement.getBoundingClientRect()

    // Can intercom be intersected with target in the initial position.
    const isIntersectedInitially = checkIsIntersectedY(
      targetRect,
      intercomRect,
      currentShiftYRef.current,
    )

    // Intercom in the initial position will not be intersected, reset his shift.
    if (!isIntersectedInitially) {
      setShiftY(0)
      return
    }

    // In other case shift intercom to the top relative to the target.
    const shiftYDelta = getIntercomShiftYDelta(targetRect, intercomRect)
    const shiftY = currentShiftYRef.current + shiftYDelta
    setShiftY(shiftY)

  }, UPDATE_INTERVAL)
}
