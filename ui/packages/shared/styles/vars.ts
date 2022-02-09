/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

// Colors.
const colorGray = '#c0c0c0'

export const colors = {
  white: '#fff',
  gray: '#D6D6D6',

  status: {
    ok: '#0db94d',
    warning: '#fd8411',
    error: '#ff2020',
    waiting: '#ffad5f',
    unknown: colorGray
  }
}

// Mixins.
export const createTransitionInteractive = (...props: string[]) =>
  props.map((prop) => `${prop} .2s ease-out`).join(',');

export const borderRadius = '4px';

export const resetStyles = {
  margin: 0,
  padding: 0,
}

export const resetStylesRoot = {
  ...resetStyles,
  '& *': {
    ...resetStyles
  }
}
