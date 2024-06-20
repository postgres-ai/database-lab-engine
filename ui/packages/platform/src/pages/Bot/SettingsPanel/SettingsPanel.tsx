import React, { useMemo } from 'react';
import cn from "classnames";
import { Button, makeStyles, useMediaQuery } from "@material-ui/core";
import SettingsOutlinedIcon from "@material-ui/icons/SettingsOutlined";
import { colors } from "@postgres.ai/shared/styles/colors";
import { theme } from "@postgres.ai/shared/styles/theme";
import { permalinkLinkBuilder } from "../utils";
import { useAiBot } from "../hooks";
import DeveloperModeIcon from "@material-ui/icons/DeveloperMode";

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
      textDecoration: 'none',
      '& > span': {
        textTransform: 'capitalize'
      }
    },
    labelVisibility: {
      marginLeft: '0.5rem',
      [theme.breakpoints.down('sm')]: {
        marginLeft: '0.25rem'
      },
      '&:hover': {
        backgroundColor: colors.secondary1.main
      }
    },
    labelModel: {
      background: colors.secondary1.main,
    },
    labelModelInvalid: {
      background: colors.state.error,
      border: "none",
      cursor: 'pointer',
      '&:hover': {
        backgroundColor: colors.primary.dark
      }
    },
    labelPrivate: {
      backgroundColor: colors.pgaiDarkGray,
    },
    disabled: {
      pointerEvents: "none"
    },
    button: {
      marginLeft: '0.5rem',
      [theme.breakpoints.down('sm')]: {
        border: 'none',
        minWidth: '2rem',
        height: '2rem',
        padding: 0,
        marginLeft: '0.25rem',
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
  const { messages, chatVisibility, aiModel, aiModelsLoading } = useAiBot();
  const permalinkId = useMemo(() => messages?.[0]?.id, [messages]);

  let modelLabel;

  if (aiModel) {
    modelLabel = (
      <span
        className={cn(classes.label, classes.labelModel)}
      >
        {aiModel.name}
      </span>
    )
  } else {
    modelLabel = (
      <button
        className={cn(classes.label, classes.labelModelInvalid)}
        onClick={onSettingsClick}
      >
        Model not set
      </button>
    )
  }

  return (
    <>
      {!aiModelsLoading && modelLabel}
      {permalinkId && <a
        href={permalinkId && chatVisibility === 'public' ? permalinkLinkBuilder(permalinkId) : ''}
        className={cn(classes.label, classes.labelVisibility,
          {
            [classes.labelPrivate]: chatVisibility === 'private',
            [classes.disabled]: chatVisibility === 'private' || !permalinkId
          }
        )}
        target="_blank"
        aria-disabled={chatVisibility === 'private' || !permalinkId}
      >
        <span>{chatVisibility}</span> thread
      </a>}
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