/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/styles';

import { colors } from '@postgres.ai/shared/styles/colors';

import { Day } from './Day';

const useStyles = makeStyles({
  root: {
    marginTop: '10px',
  },
  container: {
    display: 'flex',
    justifyContent: 'space-between',
    position: 'relative'
  },
  line: {
    margin: '0 9px -23px 9px',
    paddingTop: '22px',
    borderBottom: `1px solid ${colors.pgaiDarkGray}`
  },
  label: {
    fontSize: '12px'
  }
});

export const TimeLine = () => {
  const classes = useStyles();

  return (
    <div className={classes.root}>
      <div className={classes.line} />
      <div className={classes.container}>
        <Day text='Jun 20' />
        <Day text='Jun 21' />
        <Day text='Jun 22' />
        <Day text='Jun 23' />
        <Day text='Jun 24' />
        <Day text='Jun 25' />
        <Day text='Jun 26' />
      </div>
      <div className={classes.container}>
        <span className={classes.label}>16:33:20 UTC</span>
        <span className={classes.label}>10:30:00 UTC</span>
      </div>
    </div>
  );
};
