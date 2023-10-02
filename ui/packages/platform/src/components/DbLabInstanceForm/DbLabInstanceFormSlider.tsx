/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import Slider from '@material-ui/core/Slider'
import { makeStyles } from '@material-ui/core'

const useStyles = makeStyles({
  root: {
    width: '100%',
    marginBottom: 0,
  },
  valueLabel: {
    '& > span': {
      backgroundColor: 'transparent',
    },
    '& > span > span': {
      color: '#000',
      fontWeight: 'bold',
    },
  },
})

export const StorageSlider = ({
  value,
  onChange,
  customMarks,
  sliderOptions,
}: {
  value: number
  customMarks: { value: number; scaledValue: number; label: string | number }[]
  sliderOptions: { [key: string]: number }
  onChange: (event: React.ChangeEvent<{}>, value: unknown) => void
}) => {
  const classes = useStyles()

  const scale = (value: number) => {
      if (customMarks) {
        const previousMarkIndex = Math.floor(value / 25)
        const previousMark = customMarks[previousMarkIndex]
        const remainder = value % 25

        if (remainder === 0) {
          return previousMark?.scaledValue
        }

        const nextMark = customMarks[previousMarkIndex + 1]
        const increment = (nextMark?.scaledValue - previousMark?.scaledValue) / 25
        return remainder * increment + previousMark?.scaledValue
      } else {
        return value
      }
  }

  return (
    <Slider
      {...sliderOptions}
      value={value}
      scale={scale}
      marks={customMarks}
      onChange={onChange}
      valueLabelDisplay="auto"
      valueLabelFormat={() => value}
      aria-labelledby="non-linear-slider"
      classes={{ root: classes.root, valueLabel: classes.valueLabel }}
    />
  )
}
