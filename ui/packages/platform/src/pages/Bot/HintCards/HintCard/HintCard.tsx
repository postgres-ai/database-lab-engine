import React from 'react'
import { Hint } from 'pages/Bot/hints'
import { matchHintTypeAndIcon } from "../../utils";
import { makeStyles } from "@material-ui/core";
import { useAiBot } from "../../hooks";

const useStyles = makeStyles((theme) => ({
  container: {
    backgroundColor: 'transparent',
    border: '1px solid rgba(0, 0, 0, 0.25)',
    borderRadius: '0.5rem',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'flex-start',
    cursor: 'pointer',
    width: '11rem',
    height: '6rem',
    padding: '0.5rem',
    color: 'black',
    textAlign: 'left',
    fontSize: '0.938rem',
    transition: '0.2s ease-in',

    '& svg': {
      width: '22px',
      height: '22px',
      marginBottom: '0.5rem',
      '& path': {
        stroke: 'black',
      },
      [theme.breakpoints.down('sm')]: {
        width: '16px',
        height: '16px'
      }
    },

    '&:hover, &:focus-visible': {
      border: '1px solid rgba(0, 0, 0, 0.8)',
    },
    [theme.breakpoints.down(1024)]: {
      flex: '1 1 45%',
    },
    [theme.breakpoints.down(480)]: {
      margin: '0 0.5rem',
      fontSize: '0.813rem',
      height: 'auto',
    },
    [theme.breakpoints.down(330)]: {
      fontSize: '.75rem'
    }
  },
}));

export const HintCard = (props: Hint & {orgId: number}) => {
  const { prompt, hint, type, orgId } = props;
  const { sendMessage } = useAiBot();

  const classes = useStyles();

  const handleSendMessage = async () => {
    await sendMessage({
      content: prompt,
      org_id: orgId,
    })
  }

  return (
    <button
      onClick={handleSendMessage}
      className={classes.container}
    >
      {React.createElement(matchHintTypeAndIcon(type))}
      <span>
        {hint}
      </span>
    </button>
  )
}