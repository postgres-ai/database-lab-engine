/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { API_URL_PREFIX } from "../../config/env";
import { DebugMessage } from "../../types/api/entities/bot";
import { HintType } from "./hints";
import { FunctionComponent } from "react";
import { ArrowGrowthIcon } from "./HintCards/ArrowGrowthIcon/ArrowGrowthIcon";
import { WrenchIcon } from "./HintCards/WrenchIcon/WrenchIcon";
import { TableIcon } from "./HintCards/TableIcon/TableIcon";
import { CommonTypeIcon } from "./HintCards/CommonTypeIcon/CommonTypeIcon";

export const permalinkLinkBuilder = (id: string): string => {
  const apiUrl = process.env.REACT_APP_API_URL_PREFIX || API_URL_PREFIX;
  const isV2API = /https?:\/\/.*v2\.postgres\.ai\b/.test(apiUrl);
  return `https://${isV2API ? 'v2.' : ''}postgres.ai/chats/${id}`;
};

export const disallowedHtmlTagsForMarkdown= [
  'script',
  'style',
  'iframe',
  'form',
  'input',
  'link',
  'meta',
  'embed',
  'object',
  'applet',
  'base',
  'frame',
  'frameset',
  'audio',
  'video',
  'button',
  'select',
  'option',
  'textarea'
];

export const createMessageFragment = (messages: DebugMessage[]): DocumentFragment => {
  const fragment = document.createDocumentFragment();

  messages.forEach((item) => {
    const textBeforeLink = `[${item.created_at}]: `;
    const parts = item.content.split(/(https?:\/\/[^\s)"']+)/g);

    const messageContent = parts.map((part) => {
      if(/https?:\/\/[^\s)"']+/.test(part)) {
        const link = document.createElement('a');
        link.href = part;
        link.target = '_blank';
        link.textContent = part;
        return link;
      } else {
        return document.createTextNode(part);
      }
    });

    fragment.appendChild(document.createTextNode(textBeforeLink));
    messageContent.forEach((node) => fragment.appendChild(node));
    fragment.appendChild(document.createTextNode('\n'));
  });

  return fragment;
};

export const formatLanguageName = (language: string): string => {
  const specificCases: { [key: string]: string } = {
    "sql": "SQL",
    "pl/pgsql": "PL/pgSQL",
    "pl/python": "PL/Python",
    "json": "JSON",
    "yaml": "YAML",
    "html": "HTML",
    "xml": "XML",
    "css": "CSS",
    "csv": "CSV",
    "toml": "TOML",
    "ini": "INI",
    "r": "R",
    "php": "PHP",
    "sqlalchemy": "SQLAlchemy",
    "xslt": "XSLT",
    "xsd": "XSD",
    "ajax": "AJAX",
    "tsql": "TSQL",
    "pl/sql": "PL/SQL",
    "dax": "DAX",
    "sparql": "SPARQL"
  };

  const normalizedLanguage = language.toLowerCase();

  if (specificCases[normalizedLanguage]) {
    return specificCases[normalizedLanguage];
  } else {
    return language.charAt(0).toUpperCase() + language.slice(1).toLowerCase();
  }
}

export function matchHintTypeAndIcon(hintType: HintType): FunctionComponent<SVGElement> {
  switch (hintType) {
    case 'performance':
      return ArrowGrowthIcon
    case 'settings':
      return WrenchIcon
    case 'design':
      return TableIcon
    default:
      return CommonTypeIcon
  }
}