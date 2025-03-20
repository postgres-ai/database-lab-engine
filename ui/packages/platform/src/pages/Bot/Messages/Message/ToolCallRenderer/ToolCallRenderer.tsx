import { useState } from "react";
import { SourcesShortList } from "../../Sources/SourcesShortList";
import { SourcesFullList } from "../../Sources/SourcesFullList";
import { Box } from "@mui/material";


type MarkdownNode = {
  type: string;
  tagName: string;
  properties?: {
    ['data-json']?: string;
    dataJson?: string;
  };
  children?: MarkdownNode[];
}

type ToolCallRendererProps = {
  'data-json'?: string;
  node?: MarkdownNode;
}

export const ToolCallRenderer = (props: ToolCallRendererProps) => {
  const [isSourcesVisible, setSourcesVisible] = useState(false);

  const dataJson =
    props?.['data-json'] ||
    props?.node?.properties?.dataJson;

  if (!dataJson) {
    return null;
  }


  let parsed;
  try {
    const preparedData = JSON.parse(dataJson);

    const cleaned = preparedData.replace(/\\n/g, '').trim();

    parsed = JSON.parse(cleaned);
  } catch (err) {
    console.error("ToolCall parsing error: ", err);
    return null;
  }


  const toggleSources = () => {
    setSourcesVisible(prevState => !prevState)
  }

  return (
    <>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          marginBottom: '0.5rem',
          fontWeight: 500,
          lineHeight: 1.2
        }}
      >
        <span>Search query:&nbsp;{parsed?.[0]?.arguments?.input}</span>
        <span>Count:&nbsp;{parsed?.[0]?.arguments?.match_count}</span>
        <span>Categories:&nbsp;{parsed?.[0]?.arguments?.categories?.join(', ')}</span>
      </Box>
      <SourcesShortList
        toolCallResult={parsed}
        isVisible={isSourcesVisible}
        onChangeVisibility={toggleSources}
      />
      {isSourcesVisible && <SourcesFullList toolCallResult={parsed} />}
    </>
  );
}