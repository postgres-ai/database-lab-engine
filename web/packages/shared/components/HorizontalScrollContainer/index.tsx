/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useState, useRef, useLayoutEffect, useEffect } from 'react'
import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'

import { createTransitionInteractive, borderRadius } from '@postgres.ai/shared/styles/vars'

import { checkIsVisibleLeftCurtain, checkIsVisibleRightCurtain } from './utils'
import { Dimensions } from './types'

const useStyles = makeStyles((theme) => ({
  root: {
    position: 'relative',
    overflow: 'hidden',

    [theme.breakpoints.down('xs')]: {
      borderRadius,
    },
  },
  curtain: {
    position: 'absolute',
    top: 0,
    width: '25px',
    height: '100%',
    pointerEvents: 'none',
    transition: createTransitionInteractive('opacity'),
  },
  curtainLeft: {
    left: 0,
    background: 'linear-gradient(to right, rgba(0,0,0,0.08), transparent);',
  },
  curtainRight: {
    right: 0,
    background: 'linear-gradient(to left, rgba(0,0,0,0.08), transparent);',
  },
  curtainHidden: {
    opacity: 0,
  },
  content: {
    overflow: 'auto',
  },
}))

type Props = {
  children: React.ReactNode
  classes?: {
    root?: string
    content?: string
    curtain?: string
    curtainLeft?: string
    curtainRight?: string
  }
}

export const HorizontalScrollContainer = (props: Props) => {
  const classes = useStyles()

  const [dimensions, setDimensions] = useState<Dimensions | null>(null)

  const contentRef = useRef<HTMLDivElement>(null)

  const calcDimensions = () => {
    if (!contentRef.current) {
      return
    }

    const { offsetWidth, scrollWidth, scrollLeft } = contentRef.current

    setDimensions({
      offsetWidth,
      scrollWidth,
      scrollLeft,
    })
  }

  // Initial calculation.
  useLayoutEffect(() => {
    calcDimensions()
  }, [])

  // Listen window resizes for the next calculations.
  useEffect(() => {
    window.addEventListener('resize', calcDimensions)
    return () => window.removeEventListener('resize', calcDimensions)
  }, [])

  const isVisibleLeft = dimensions && checkIsVisibleLeftCurtain(dimensions)
  const isVisibleRight = dimensions && checkIsVisibleRightCurtain(dimensions)

  return (
    <div className={clsx(classes.root, props.classes?.root)}>
      <div
        className={clsx(
          classes.curtain,
          classes.curtainLeft,
          !isVisibleLeft && classes.curtainHidden,
          props.classes?.curtain,
          props.classes?.curtainLeft,
        )}
      />
      <div
        ref={contentRef}
        className={clsx(classes.content, props.classes?.content)}
        onScroll={calcDimensions}
      >
        {props.children}
      </div>
      <div
        className={clsx(
          classes.curtain,
          classes.curtainRight,
          !isVisibleRight && classes.curtainHidden,
          props.classes?.curtain,
          props.classes?.curtainRight,
        )}
      />
    </div>
  )
}
