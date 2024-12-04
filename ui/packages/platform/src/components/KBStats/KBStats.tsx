import React, { useMemo } from "react";
import { useKBStats } from "./hooks";
import Box from "@mui/material/Box/Box";
import { makeStyles, Typography } from "@material-ui/core";
import { Link } from "@postgres.ai/shared/components/Link2";

const useStyles = makeStyles((theme) => ({
  container: {
    marginTop: 42,
    '& p': {
      margin: 0,
      lineHeight: 1.5,
    },
    [theme.breakpoints.down(480)]: {
      marginTop: 24
    },
    [theme.breakpoints.down(360)]: {
      marginTop: 0
    }
  },
  headingLink: {
    fontSize: 16,
    [theme.breakpoints.down(330)]: {
      fontSize: 14
    }
  },
}))

export const KBStats = () => {
  const { data, loading, error } = useKBStats();
  const classes = useStyles()

  const { totalSum, lastUpdate } = useMemo(() => {
    if (!data?.length) {
      return { totalSum: 0, lastUpdate: '' };
    }

    const categoryTotals = new Map<string, number>();
    let latestDate = data[0].last_document_date;

    data.forEach(({ category, total_count, last_document_date }) => {
      categoryTotals.set(category, total_count);
      if (new Date(last_document_date) > new Date(latestDate)) {
        latestDate = last_document_date;
      }
    });

    latestDate = new Date(latestDate).toISOString().replace('T', ' ').split('.')[0]

    const totalSum = Array.from(categoryTotals.values()).reduce((sum, count) => sum + count, 0);
    return { totalSum, lastUpdate: latestDate };
  }, [data]);

  if (error || loading || !data?.length) {
    return <div className={classes.container} style={{ height: 58.5 }}></div>;
  }

  return (
    <Box className={classes.container}>
      <p>Knowledge base contains {totalSum.toLocaleString(navigator.language)} documents.</p>
      <p>Last updated: {lastUpdate}.</p>
      <Link
          external
          to={`https://postgres.ai/docs/reference-guides/postgres-ai-bot-reference#tool-rag_search`}
          target="_blank"
          title="Show full information"
        >
        Details
      </Link>
    </Box>
  );
}