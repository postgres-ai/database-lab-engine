import React, { useState } from 'react';
import { Accordion, AccordionDetails, AccordionSummary, Typography, makeStyles, Button } from '@material-ui/core';
import { Prism as SyntaxHighlighter, SyntaxHighlighterProps } from 'react-syntax-highlighter'
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import languages from 'react-syntax-highlighter/dist/esm/languages/prism/supported-languages';
import ExpandMoreIcon from "@material-ui/icons/ExpandMore";
import FileCopyOutlinedIcon from '@material-ui/icons/FileCopyOutlined';
import CheckCircleOutlineIcon from '@material-ui/icons/CheckCircleOutline';

const useStyles = makeStyles((theme) => ({
  container: {
    marginTop: '.5em',
    width: '100%'
  },
  header: {
    background: 'rgb(240, 240, 240)',
    borderTopLeftRadius: 8,
    borderTopRightRadius: 8,
    padding: '.2rem 1rem',
    display: 'flex'
  },
  languageName: {
    fontSize: '0.813rem',
    color: theme.palette.text.primary
  },
  copyButton: {
    marginLeft: 'auto',
    color: theme.palette.text.primary,
    padding: '0',
    minHeight: 'auto',
    fontSize: '0.813rem',
    border: 0,
    '&:hover': {
      backgroundColor: 'transparent'
    }
  },
  copyButtonIcon: {
    width: '0.813rem',
  },
  summary: {
    textDecoration: 'underline',
    textDecorationStyle: 'dotted',
    cursor: 'pointer',
    backgroundColor: 'transparent',
    boxShadow: 'none',
    display: 'inline-flex',
    minHeight: '32px!important',
    padding: 0,
    '&:hover': {
      textDecoration: 'none'
    }
  },
  details: {
    padding: 0,
    backgroundColor: 'transparent'
  },
  accordion: {
    boxShadow: 'none',
    backgroundColor: 'transparent',
  },
  pre: {
    width: '100%',
    marginTop: '0!important',

  }
}));

export const CodeBlock = ({ value, language }: { value: string, language?: string | null }) => {
  const classes = useStyles();
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);

  const codeLines = value.split('\n');
  const handleToggle = () => setExpanded(!expanded);


  const isValidLanguage = language && languages.includes(language);

  const syntaxHighlighterProps: SyntaxHighlighterProps = {
    showLineNumbers: true,
    language: language || 'sql',
    style: oneLight,
    className: classes.pre,
    children: value
  }

  const handleCopy = () => {
    if ('clipboard' in navigator) {
      navigator.clipboard.writeText(value).then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      });
    }
  };


  const header = (
    <div className={classes.header}>
      {isValidLanguage && <Typography className={classes.languageName}>{language}</Typography>}
      <Button
        size="small"
        variant="outlined"
        className={classes.copyButton}
        onClick={handleCopy}
        startIcon={copied
          ? <CheckCircleOutlineIcon className={classes.copyButtonIcon} />
          : <FileCopyOutlinedIcon className={classes.copyButtonIcon} />
        }
      >
        {copied ? 'Copied' : 'Copy'}
      </Button>
    </div>
  )

  if (codeLines.length > 20) {
    return (
      <Accordion expanded={expanded} onChange={handleToggle} className={classes.accordion}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />} className={classes.summary}>
          <Typography>{expanded ? 'Hide' : 'Show'} code block ({codeLines.length} LOC)</Typography>
        </AccordionSummary>
        <AccordionDetails className={classes.details}>
          <div className={classes.container}>
            {header}
            <SyntaxHighlighter
              {...syntaxHighlighterProps}
              children={value}
            />
          </div>
        </AccordionDetails>
      </Accordion>
    );
  }

  return (
    <div className={classes.container}>
      {header}
      <SyntaxHighlighter
        {...syntaxHighlighterProps}
        children={value}
      />
    </div>
  );
}