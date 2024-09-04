import IconButton from "@material-ui/core/IconButton";
import { ZoomInRounded, ZoomOutRounded, SaveAltRounded, FileCopyOutlined } from "@material-ui/icons";
import {  makeStyles } from "@material-ui/core";
import React, { useCallback } from "react";
import Divider from "@material-ui/core/Divider";

const useStyles = makeStyles(
  () => ({
    container: {
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',

      position: 'absolute',
      bottom: 20,
      right: 10,
      zIndex: 2,
    },
    controlButtons: {
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',

      border: '1px solid rgba(0, 0, 0, 0.12)',
      borderRadius: 8,

      background: 'white',

      "& .MuiIconButton-root": {
        fontSize: '1.5rem',
        color: 'rgba(0, 0, 0, 0.72)',
        padding: 8,
        '&:hover': {
          color: 'rgba(0, 0, 0, 0.95)',
        },
        '&:first-child': {
          borderRadius: '8px 8px 0 0',
        },
        '&:last-child': {
          borderRadius: ' 0 0 8px 8px',
        }
      }
    },
    divider: {
      width: 'calc(100% - 8px)',
    },
    actionButton: {
      fontSize: '1.5rem',
      color: 'rgba(0, 0, 0, 0.72)',
      padding: 8,
      marginBottom: 8,
      '&:hover': {
        color: 'rgba(0, 0, 0, 0.95)',
      },
    }
  }))


type MermaidDiagramControlsProps = {
  handleZoomIn: () => void,
  handleZoomOut: () => void,
  diagramRef:  React.RefObject<HTMLDivElement>,
  sourceCode: string
}

export const MermaidDiagramControls = (props: MermaidDiagramControlsProps) => {
  const { sourceCode, handleZoomOut, handleZoomIn, diagramRef } = props;
  const classes = useStyles();

  const handleSaveClick = useCallback(() => {
    if (diagramRef.current) {
      const svgElement = diagramRef.current.querySelector('svg');
      if (svgElement) {
        const svgData = new XMLSerializer().serializeToString(svgElement);
        const svgBlob = new Blob([svgData], { type: 'image/svg+xml;charset=utf-8' });
        const url = URL.createObjectURL(svgBlob);

        const link = document.createElement('a');
        link.href = url;
        link.download = 'er-diagram.svg';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        URL.revokeObjectURL(url);
      }
    }
  }, []);

  const handleCopyClick = async () => {
    if ('clipboard' in navigator) {
      await navigator.clipboard.writeText(sourceCode);
    }
  }

  return (
    <div className={classes.container}>
      <IconButton
        title="Copy contents"
        aria-label="Copy contents"
        className={classes.actionButton}
        onClick={handleCopyClick}
      >
        <FileCopyOutlined />
      </IconButton>
      <IconButton
        title="Download as SVG"
        aria-label="Download diagram as SVG"
        className={classes.actionButton}
        onClick={handleSaveClick}
      >
        <SaveAltRounded />
      </IconButton>

      <div className={classes.controlButtons}>
        <IconButton
          onClick={handleZoomIn}
          title="Zoom In"
          aria-label="Zoom In"
        >
          <ZoomInRounded />
        </IconButton>
        <Divider className={classes.divider} />
        <IconButton
          onClick={handleZoomOut}
          title="Zoom Out"
          aria-label="Zoom Out"
        >
          <ZoomOutRounded />
        </IconButton>
      </div>
    </div>
  )
}