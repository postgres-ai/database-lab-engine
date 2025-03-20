import React, { useState } from "react";
import { Button } from "@postgres.ai/shared/components/Button2";
import ExpandMoreIcon from "@material-ui/icons/ExpandMore";
import { CardContent, Collapse } from "@mui/material";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

type ThinkBlockProps = {
  'data-think'?: string;
  node?: {
    properties?: {
      'data-think'?: string;
      dataThink?: string;
    };
  };
}

type ThinkingCardProps = {
  content: string;
}

const ThinkingCard = ({ content }: ThinkingCardProps) => {
  const [expanded, setExpanded] = useState(true);
  // TODO: Add "again"
  // TODO: Replace with "reasoned for X seconds"
  return (
    <>
      <Button
        onClick={() => setExpanded(!expanded)}
      >
        Took a moment to think
        <ExpandMoreIcon />
      </Button>

      <Collapse in={expanded}>
        <CardContent>
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {content}
          </ReactMarkdown>
        </CardContent>
      </Collapse>
    </>
  )
}

export const ThinkBlockRenderer = React.memo((props: ThinkBlockProps) => {
  const dataThink =
    props?.['data-think'] ||
    props?.node?.properties?.['data-think'] ||
    props?.node?.properties?.dataThink;

  if (!dataThink) return null;

  let rawText = '';
  try {
    rawText = JSON.parse(dataThink);
  } catch (err) {
    console.error('Failed to parse data-think JSON:', err);
  }

  return (
    <ThinkingCard content={rawText}/>
  )
}, (prevProps, nextProps) => {
  return prevProps['data-think'] === nextProps['data-think'];
})