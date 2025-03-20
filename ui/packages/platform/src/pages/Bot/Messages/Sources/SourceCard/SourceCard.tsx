import React from 'react';
import cn from 'classnames';
import { makeStyles } from "@material-ui/core";
import { colors } from "@postgres.ai/shared/styles/colors";
import { ArrowDropDown, ArrowDropDownOutlined, KeyboardArrowDown, KeyboardArrowUp } from "@material-ui/icons";

const useStyles = makeStyles((theme) => ({
  shortContainer: {
    backgroundColor: 'transparent',
    border: '1px solid rgba(0, 0, 0, 0.25)',
    borderRadius: '0.5rem',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'flex-start',
    cursor: 'pointer',
    width: '8rem',
    height: '5rem',
    padding: '0.5rem',
    color: 'black',
    textAlign: 'left',
    fontSize: '0.938rem',
    transition: '0.2s ease-in',
    textDecoration: "none",
    overflow: 'hidden',
    '&:hover, &:focus-visible': {
      border: '1px solid rgba(0, 0, 0, 0.8)',
    },
    [theme.breakpoints.down(330)]: {
      fontSize: '.75rem'
    },
  },
  fullContainer: {
    width: '100%',
    height: 'auto',
    border: 'none!important',
  },
  showMoreContainer: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    fontSize: '1.5rem',
    color: colors.pgaiDarkGray,
    width: '2rem',
  },
  link: {
    fontSize: '0.688rem',
    marginBottom: 4,
    color: colors.pgaiDarkGray
  },
  content: {
    fontSize: '0.75rem',
    display: '-webkit-box',
    '-webkit-line-clamp': 3,
    '-webkit-box-orient': 'vertical',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    wordWrap: 'break-word',
    overflowWrap: 'break-word',
  },
  title: {
    fontSize: '1rem',
    display: '-webkit-box',
    '-webkit-line-clamp': 2,
   ' -webkit-box-orient': 'vertical',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    fontWeight: 500
  },
  fullListCardContent: {
    fontSize: '0.875rem',
    marginTop: 4,
  }
}));

type SourceCardProps = {
  title?: string;
  content?: string;
  url?: string;
  variant: 'shortListCard' | 'fullListCard' | 'showMoreCard',
  isVisible?: boolean;
  onShowFullListClick?: () => void;
}

export const SourceCard = (props: SourceCardProps) => {
  const { title, content, url, variant, isVisible, onShowFullListClick } = props;
  const classes = useStyles();

  if (variant === 'shortListCard') {
    return (
      <a href={url} target="_blank" rel="noopener noreferrer" className={classes.shortContainer}>
        <span className={classes.link}>
          {new URL(url || '').hostname}
        </span>
        <span className={classes.content}>
          {title}
        </span>
      </a>
    )
  } else if (variant === 'fullListCard') {
    return (
      <a href={url} target="_blank" rel="noopener noreferrer" className={cn(classes.shortContainer, classes.fullContainer)}>
        <span className={classes.link}>
          {new URL(url || '').hostname}
        </span>
        <span className={classes.title}>
          {title}
        </span>
        <span className={cn(classes.content, classes.fullListCardContent)}>
          {content}
        </span>
      </a>
    )
  } else if (variant === 'showMoreCard') {
    return (
      <button
        onClick={onShowFullListClick}
        className={cn(classes.shortContainer, classes.showMoreContainer)}
        title={isVisible ? "Hide full list" : "Show full list"}
      >
        {isVisible ? <KeyboardArrowUp /> : <KeyboardArrowDown />}
      </button>
    )
  } else {
    return null;
  }
}