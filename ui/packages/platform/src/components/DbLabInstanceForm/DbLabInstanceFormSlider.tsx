/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import Slider from '@material-ui/core/Slider'
import { makeStyles } from '@material-ui/core'

const storageMarks = [
  {
    value: 30,
    label: '30 GiB',
    scaledValue: 30,
  },
  {
    value: 500,
    label: '500 GiB',
    scaledValue: 500,
  },
  {
    value: 1000,
    label: '1000 GiB',
    scaledValue: 1000,
  },
  {
    value: 1500,
    label: '1500 GiB',
    scaledValue: 1500,
  },
  {
    value: 2000,
    label: '2000 GiB',
    scaledValue: 2000,
  },
]

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

const scale = (value: number) => {
  const previousMarkIndex = Math.floor(value / 25)
  const previousMark = storageMarks[previousMarkIndex]
  const remainder = value % 25

  if (remainder === 0) {
    return previousMark?.scaledValue
  }

  const nextMark = storageMarks[previousMarkIndex + 1]
  const increment = (nextMark?.scaledValue - previousMark?.scaledValue) / 25
  return remainder * increment + previousMark?.scaledValue
}

export const StorageSlider = ({
  value,
  onChange,
}: {
  value: number
  onChange: (event: React.ChangeEvent<{}>, value: unknown) => void
}) => {
  const classes = useStyles()

  return (
    <Slider
      classes={{ root: classes.root, valueLabel: classes.valueLabel }}
      value={value}
      min={30}
      step={10}
      max={2000}
      marks={storageMarks}
      scale={scale}
      onChange={onChange}
      valueLabelFormat={() => value}
      valueLabelDisplay="auto"
      aria-labelledby="non-linear-slider"
    />
  )
}
