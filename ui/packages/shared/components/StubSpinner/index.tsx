/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/core';
import clsx from 'clsx'

import { Spinner, Props as SpinnerProps } from '@postgres.ai/shared/components/Spinner'

import { colors } from '@postgres.ai/shared/styles/colors';

type Props = {
  className?: string,
  size?: SpinnerProps['size']
}

const useStyles = makeStyles({
  root: {
    position: 'absolute',
    top: 0,
    left: 0,
    height: '100%',
    width: '100%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: colors.white
  }
});

export const StubSpinner = (props: Props) => {
  const { className, size = 'lg' } = props
  const classes = useStyles();

  return (
    <div className={clsx(classes.root, className)}>
      <Spinner size={size}/>
    </div>
  );
};
