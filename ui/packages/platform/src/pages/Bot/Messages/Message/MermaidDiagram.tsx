import React, { useCallback, useEffect, useRef, useState } from 'react';
import mermaid from 'mermaid';
import { makeStyles } from "@material-ui/core";
import { MermaidDiagramControls } from "./MermaidDiagramControls";
import cn from "classnames";

type MermaidDiagramProps = {
  chart: string
}

type DiagramPosition = {
  x: number,
  y: number
}

type DiagramState = {
  scale: number,
  position: DiagramPosition,
  startPosition: DiagramPosition,
  dragging: boolean
}

const useStyles = makeStyles(
  (theme) => ({
    container: {
      position: 'relative',
      width: '100%',
      overflow: 'hidden'
    },
    mermaid: {
      [theme.breakpoints.up('sm')]: {
        display: 'flex',
        justifyContent: 'center',
      }
    },
  }))

mermaid.initialize({ startOnLoad: true, er: {  useMaxWidth: false } });

export const MermaidDiagram = React.memo((props: MermaidDiagramProps) => {
  const { chart } = props;

  const classes = useStyles();

  // Consolidated state management
  const [diagramState, setDiagramState] = useState<DiagramState>({
    scale: 1,
    position: { x: 0, y: 0 },
    dragging: false,
    startPosition: { x: 0, y: 0 },
  });

  const [isDiagramValid, setDiagramValid] = useState<boolean | null>(null);
  const [diagramError, setDiagramError] = useState<string | null>(null)

  const diagramRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let isMounted = true;
    if (isDiagramValid === null || chart) {
      mermaid.parse(chart)
        .then(() => {
          if (isMounted) {
            setDiagramValid(true);
            mermaid.contentLoaded();
          }
        })
        .catch((e) => {
          if (isMounted) {
            setDiagramValid(false);
            setDiagramError(e.message)
            console.error('Diagram contains errors:', e.message);
          }
        });
    }

    return () => {
      isMounted = false;
    };
  }, [chart, isDiagramValid]);

  const handleZoomIn = useCallback(() => {
    setDiagramState((prev) => ({
      ...prev,
      scale: Math.min(prev.scale + 0.1, 2),
    }));
  }, []);

  const handleZoomOut = useCallback(() => {
    setDiagramState((prev) => ({
      ...prev,
      scale: Math.max(prev.scale - 0.1, 0.8),
    }));
  }, []);

  const handleMouseDown = useCallback((event: React.MouseEvent) => {
    setDiagramState((prev) => ({
      ...prev,
      dragging: true,
      startPosition: { x: event.clientX - prev.position.x, y: event.clientY - prev.position.y },
    }));
  }, []);

  const handleMouseMove = useCallback((event: React.MouseEvent) => {
    if (diagramState.dragging) {
      setDiagramState((prev) => ({
        ...prev,
        position: { x: event.clientX - prev.startPosition.x, y: event.clientY - prev.startPosition.y },
      }));
    }
  }, [diagramState.dragging]);

  const handleMouseUp = useCallback(() => {
    setDiagramState((prev) => ({ ...prev, dragging: false }));
  }, []);

  const handleTouchStart = useCallback((event: React.TouchEvent) => {
    const touch = event.touches[0];
    setDiagramState((prev) => ({
      ...prev,
      dragging: true,
      startPosition: { x: touch.clientX - prev.position.x, y: touch.clientY - prev.position.y },
    }));
  }, []);

  const handleTouchMove = useCallback((event: React.TouchEvent) => {
    if (diagramState.dragging) {
      const touch = event.touches[0];
      setDiagramState((prev) => ({
        ...prev,
        position: { x: touch.clientX - prev.startPosition.x, y: touch.clientY - prev.startPosition.y },
      }));
    }
  }, [diagramState.dragging]);

  const handleTouchEnd = useCallback(() => {
    setDiagramState((prev) => ({ ...prev, dragging: false }));
  }, []);

  if (isDiagramValid === null) {
    return <p>Validating diagram...</p>;
  }

  if (isDiagramValid) {
    return (
      <div className={classes.container}>
        <div
          className={cn("mermaid", classes.mermaid)}
          ref={diagramRef}
          style={{
            transform: `scale(${diagramState.scale}) translate(${diagramState.position.x}px, ${diagramState.position.y}px)`,
            transformOrigin: '50% 50%',
            cursor: diagramState.dragging ? 'grabbing' : 'grab',
          }}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
        >
          {chart}
        </div>
        <MermaidDiagramControls
          handleZoomIn={handleZoomIn}
          handleZoomOut={handleZoomOut}
          diagramRef={diagramRef}
          sourceCode={chart}
        />
      </div>
    );
  } else {
    return <p>{diagramError}</p>;
  }
});