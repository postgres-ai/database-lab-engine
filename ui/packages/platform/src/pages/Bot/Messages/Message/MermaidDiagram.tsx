import React, { useEffect } from 'react';
import mermaid from 'mermaid';

type MermaidDiagramProps = {
  chart: string
}

export const MermaidDiagram = React.memo((props: MermaidDiagramProps) => {
  const { chart } = props;
  mermaid.initialize({ startOnLoad: true });
  useEffect(() => {
    mermaid.contentLoaded();
  }, [chart]);
  return <div className="mermaid">{chart}</div>;
})