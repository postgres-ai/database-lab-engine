import React, { useMemo } from 'react';
import cn from "classnames";
import { Button, makeStyles, useMediaQuery } from "@material-ui/core";
import SettingsOutlinedIcon from "@material-ui/icons/SettingsOutlined";
import { colors } from "@postgres.ai/shared/styles/colors";
import { theme } from "@postgres.ai/shared/styles/theme";
import { Link } from "react-router-dom";
import { permalinkLinkBuilder } from "../utils";
import { useAiBot } from "../hooks";
import DeveloperModeIcon from "@material-ui/icons/DeveloperMode";
import IconButton from "@material-ui/core/IconButton";

export type SettingsPanelProps = {
  onSettingsClick: () => void;
  onConsoleClick: () => void;
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
      textDecoration: 'none'
    },
    labelPrivate: {
      backgroundColor: colors.pgaiDarkGray,
    },
    disabled: {
      pointerEvents: "none"
    },
    button: {
      marginLeft: 8,
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

export const SettingsPanel = (props: SettingsPanelProps) => {
  const { onSettingsClick, onConsoleClick } = props;
  const classes = useStyles();
  const matches = useMediaQuery(theme.breakpoints.down('sm'));

  const { messages, chatVisibility } = useAiBot();
  const permalinkId = useMemo(() => messages?.[0]?.id, [messages]);

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
      <Button
        variant="outlined"
        onClick={onConsoleClick}
        startIcon={<DeveloperModeIcon />}
        className={classes.button}
      >
        {!matches && 'Console'}
      </Button>
    </>
  )
}