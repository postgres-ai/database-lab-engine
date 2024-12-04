import React from 'react'
import { hints } from '../hints'
import { HintCard } from "./HintCard/HintCard";
import { makeStyles } from "@material-ui/core";

const useStyles = makeStyles((theme) => ({
  container: {
    display: 'flex',
    flexWrap: 'wrap',
    justifyContent: 'space-around',
    gap: '.5rem',
    marginTop: '2rem',
    [theme.breakpoints.down(1200)]: {
      justifyContent: 'center',
    },
    [theme.breakpoints.down(480)]: {
      marginBottom: '1rem',
    },
    [theme.breakpoints.down(380)]: {
      marginTop: '1rem',
      marginBottom: '.5rem',
    },
    [theme.breakpoints.down(760)]: {
      '& > *:nth-child(n+3)': {
        display: 'none',
      },
    },
  },
}));

export const HintCards = React.memo(({orgId}: {orgId: number}) => {
  const classes = useStyles();
  return (
    <div className={classes.container}>
      {
        hints.map((hint) => <HintCard key={hint.hint} {...hint} orgId={orgId} />)
      }
    </div>
  )
})