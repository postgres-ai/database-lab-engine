import React from 'react';
import { FormControl, Select, MenuItem, Typography, InputLabel, useMediaQuery } from "@mui/material";
import { SelectChangeEvent } from "@mui/material/Select";

import { useAiBot } from "../hooks";

export const ModelSelector = () => {
  const { aiModel, aiModels, setAiModel } = useAiBot();
  const isSmallScreen = useMediaQuery("(max-width: 960px)");

  const handleChange = (event: SelectChangeEvent<string | null>) => {
    const [vendor, name] = (event.target.value as string).split("/");
    const model = aiModels?.find(
      (model) => model.vendor === vendor && model.name === name
    );
    if (model) setAiModel(model);
  };

  const truncateText = (text: string, maxLength: number) => {
    return text.length > maxLength ? text.substring(0, maxLength) + "..." : text;
  };

  return (
    <FormControl
      variant="outlined"
      size="small"
      sx={{ minWidth: isSmallScreen ? 120 : 200 }}
    >
      <Select
        labelId="model-select-label"
        id="model-select"
        value={aiModel ? `${aiModel.vendor}/${aiModel.name}` : ""}
        onChange={handleChange}
        displayEmpty
        inputProps={{
          "aria-describedby": "Select the AI model to be used for generating responses. Different models may vary in performance. Choose the one that best suits your needs.",
          sx: {
            height: "32px",
            fontSize: "0.875rem",
            padding: isSmallScreen ? "8px 24px 8px 8px!important" : "8px 14px",
          },
        }}
        sx={{ height: "32px" }}
        renderValue={(selected) => {
          if (!selected) return "Select Model";
          const [vendor, name] = selected.split("/");
          return truncateText(`${vendor}/${name}`, isSmallScreen ? 20 : 30);
        }}
      >
        {aiModels &&
          aiModels.map((model) => (
            <MenuItem
              key={`${model.vendor}/${model.name}`}
              value={`${model.vendor}/${model.name}`}
              title={`${model.vendor}/${model.name}`}
              sx={{
                display: "flex",
                flexDirection: "column",
                alignItems: "flex-start"
              }}
            >
              <span>{truncateText(`${model.vendor}/${model.name}`, isSmallScreen ? 33 : 40)}</span>
              {model.comment && (
                <Typography
                  variant="body2"
                  color="textSecondary"
                  sx={{
                    fontSize: "0.7rem",
                    color: "rgba(0, 0, 0, 0.6)",
                  }}
                  aria-hidden="true"
                >
                  {model.comment}
                </Typography>
              )}
            </MenuItem>
          ))}
      </Select>
    </FormControl>
  );
};
