import React from 'react';
import cn from "classnames";
import { Button, makeStyles, useMediaQuery } from "@material-ui/core";
import SettingsOutlinedIcon from "@material-ui/icons/SettingsOutlined";
import { colors } from "@postgres.ai/shared/styles/colors";
import { theme } from "@postgres.ai/shared/styles/theme";
import { Link } from "react-router-dom";
import { permalinkLinkBuilder } from "../utils";

type SettingsWithLabelProps = {
  chatVisibility: 'private' | 'public';
  onSettingsClick: () => void;
  permalinkId?: string
}

const useStyles = makeStyles((theme) => ({
    label: {
      backgroundColor: colors.primary.main,
      color: colors.primary.contrastText,
      display: 'inline-block',
      borderRadius: 3,
      fontSize: 10,
      lineHeight: '12px',
      padding: 2,
      paddingLeft: 3,
      paddingRight: 3,
      verticalAlign: 'text-top',
      marginRight: 8,
      textDecoration: 'none'
    },
    labelPrivate: {
      backgroundColor: colors.pgaiDarkGray,
    },
    disabled: {
      pointerEvents: "none"
    },
    button: {
      [theme.breakpoints.down('sm')]: {
        border: 'none',
        minWidth: '2rem',
        height: '2rem',
        padding: 0,
        marginLeft: '0.5rem',
        '& .MuiButton-startIcon': {
          margin: 0
        }
      }
    }
  }),
)

export const SettingsWithLabel = (props: SettingsWithLabelProps) => {
  const { chatVisibility, onSettingsClick, permalinkId } = props;
  const classes = useStyles();
  const matches = useMediaQuery(theme.breakpoints.down('sm'));
  return (
    <>
      <a
        href={permalinkId && chatVisibility === 'public' ? permalinkLinkBuilder(permalinkId) : ''}
        className={cn(classes.label, {[classes.labelPrivate]: chatVisibility === 'private', [classes.disabled]: chatVisibility === 'private' || !permalinkId})}
        target="_blank"
        aria-disabled={chatVisibility === 'private' || !permalinkId}
      >
        This thread is {chatVisibility}
      </a>
      <Button
        variant="outlined"
        onClick={onSettingsClick}
        startIcon={<SettingsOutlinedIcon />}
        className={classes.button}
      >
        {!matches && 'Settings'}
      </Button>
    </>
  )
}