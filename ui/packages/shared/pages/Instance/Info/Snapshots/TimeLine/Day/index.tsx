/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/styles';

import { colors } from '@postgres.ai/shared/styles/colors';

type Props = {
  text: string;
};

const useStyles = makeStyles({
  root: {
    'flex': '0 1 auto',
    'display': 'flex',
    'flexDirection': 'column',
    'alignItems': 'center',
    '&:first-child': {
      'alignItems': 'flex-start',

      '& $division': {
        marginLeft: '9px'
      },

      '& $date': {
        color: colors.black
      }
    },
    '&:last-child': {
      'alignItems': 'flex-end',

      '& $division': {
        marginRight: '9px'
      },

      '& $date': {
        color: colors.black
      }
    }
  },
  point: {
    width: '19px',
    flex: '0 0 19px',
    borderRadius: '50%',
    background: colors.secondary2.lightLight,
    border: `1px solid ${colors.secondary2.main}`
  },
  division: {
    marginTop: '4px',
    flex: '0 0 4px',
    width: '1px',
    background: colors.pgaiDarkGray
  },
  date: {
    fontSize: '10px',
    marginTop: '4px',
    color: colors.pgaiDarkGray
  }
});

export const Day = (props: Props) => {
  const { text } = props;
  const classes = useStyles();

  return (
    <div className={classes.root}>
      <div className={classes.point} />
      <div className={classes.division} />
      <span className={classes.date}>{text}</span>
    </div>
  );
};
