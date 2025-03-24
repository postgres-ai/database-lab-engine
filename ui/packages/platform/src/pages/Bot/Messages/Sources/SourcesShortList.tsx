import React, { useMemo } from 'react';
import Box from "@mui/material/Box/Box";
import { SourceCard } from "./SourceCard/SourceCard";
import { ToolCallDataItem, ToolCallResultItem } from "../../../../types/api/entities/bot";
import { useMediaQuery } from '@mui/material';

type SourcesShortListProps = {
  toolCallResult: ToolCallResultItem[]
  isVisible: boolean
  onChangeVisibility: () => void
}


export const SourcesShortList = (props: SourcesShortListProps) => {
  const { toolCallResult, isVisible, onChangeVisibility } = props
  const isMobile = useMediaQuery('@media (max-width: 760px)')

  const sortedData = useMemo(() => {
    if (!toolCallResult) return []

    let aggregated: ToolCallDataItem[] = []
    toolCallResult.forEach(item => {
      if (item?.function_name === 'rag_search') {
        aggregated = aggregated.concat(item.data)
      }
    })

    aggregated.sort((a, b) => b.similarity - a.similarity)

    return aggregated
  }, [toolCallResult])

  const visibleCount = isMobile ? 2 : 4
  const visibleItems = sortedData.slice(0, visibleCount)

  return (
    <Box display="flex" marginTop="8px">
      {visibleItems.map((source, index) => (
        <Box marginRight="4px" key={index}>
          <SourceCard
            title={source.title}
            content={source.content}
            url={source.url}
            variant="shortListCard"
          />
        </Box>
      ))}

      {sortedData.length > visibleCount && (
        <SourceCard
          variant="showMoreCard"
          isVisible={isVisible}
          onShowFullListClick={onChangeVisibility}
        />
      )}
    </Box>
  )
}