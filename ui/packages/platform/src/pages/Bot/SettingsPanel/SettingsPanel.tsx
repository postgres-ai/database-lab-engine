import React, { useMemo } from 'react';
import cn from "classnames";
import { Button, makeStyles, useMediaQuery } from "@material-ui/core";
import SettingsOutlinedIcon from "@material-ui/icons/SettingsOutlined";
import { colors } from "@postgres.ai/shared/styles/colors";
import { theme } from "@postgres.ai/shared/styles/theme";
import { permalinkLinkBuilder } from "../utils";
import { useAiBot } from "../hooks";
import DeveloperModeIcon from "@material-ui/icons/DeveloperMode";
import { ModelSelector } from "../ModelSelector/ModelSelector";
import { Skeleton } from "@mui/material";

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
      marginRight: '0.5rem',
      [theme.breakpoints.down('sm')]: {
        marginRight: '0.25rem'
      },
      '&:hover': {
        backgroundColor: colors.secondary1.main
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
  const { loading } = useAiBot()
  const classes = useStyles();
  const matches = useMediaQuery(theme.breakpoints.down('sm'));
  const { messages, chatVisibility, aiModelsLoading } = useAiBot();
  const permalinkId = useMemo(() => messages?.[0]?.id, [messages]);

  return (
    <>
      {permalinkId && <>
        {loading
          ? <Skeleton variant="rectangular" className={cn(classes.label, classes.labelVisibility)} width={64} height={16} />
          : <a
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
            </a>
        }
      </>}
      {!aiModelsLoading && <ModelSelector />}
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