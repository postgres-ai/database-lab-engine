/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { API_URL_PREFIX } from "../../config/env";
import { DEFAULT_MODEL, Model } from "./hooks";

export const permalinkLinkBuilder = (id: string): string => {
  const apiUrl = process.env.REACT_APP_API_URL_PREFIX || API_URL_PREFIX;
  const isV2API = /https?:\/\/.*v2\.postgres\.ai\b/.test(apiUrl);
  return `https://${isV2API ? 'v2.' : ''}postgres.ai/chats/${id}`;
};


export const aiModelBuilder = (model: string): Model => {
  return /oai|gpt/.test(model) ? "oai" : /gemini|gcp/.test(model) ? "gcp" : DEFAULT_MODEL;
}