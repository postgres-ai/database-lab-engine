import { Box } from '@mui/material';
import { Button } from '@postgres.ai/shared/components/Button2';
import React, { useMemo, useState } from 'react'
import { ToolCallDataItem, ToolCallResultItem } from "../../../../types/api/entities/bot";
import { SourceCard } from './SourceCard/SourceCard';


type SourcesFullListProps = {
  toolCallResult: ToolCallResultItem[]
}

const INITIAL_COUNT = 10;

export const SourcesFullList = (props: SourcesFullListProps) => {
  const { toolCallResult } = props;

  const [visibleCount, setVisibleCount] = useState(INITIAL_COUNT);

  const sortedData = useMemo(() => {
    if (!toolCallResult) return [];

    const aggregated: ToolCallDataItem[] = [];

    toolCallResult.forEach(item => {
      if (item?.function_name === 'rag_search') {
        aggregated.push(...item.data);
      }
    });

    const uniqueItemsMap = new Map<string, ToolCallDataItem>();

    aggregated.forEach(item => {
      if (item.url && !uniqueItemsMap.has(item.url)) {
        uniqueItemsMap.set(item.url, item);
      }
    });

    return Array.from(uniqueItemsMap.values())
      .sort((a, b) => b.similarity - a.similarity);

  }, [toolCallResult]);

  const handleShowMore = () => {
    setVisibleCount((prev) => prev + INITIAL_COUNT);
  };

  const visibleItems = sortedData.slice(0, visibleCount);

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        width: 'min(100%, 37.25rem)',
        marginTop: '0.5rem'
      }}
    >
      {visibleItems.map((source) => (
        <Box
          sx={{
            marginBottom: '0.5rem',
            width: '100%'
          }}
          key={source.url}
        >
          <SourceCard
            title={source.title || ''}
            content={source.content || ''}
            url={source.url || ''}
            variant="fullListCard"
          />
        </Box>
      ))}

      {visibleCount < sortedData.length && (
        <Button theme="primary" onClick={handleShowMore}>Show more</Button>
      )}
    </Box>
  )
}