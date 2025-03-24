import {BotMessage} from "../../../types/api/entities/bot";

export const getMaxScrollTop = (element: HTMLElement) =>
  element.scrollHeight - element.clientHeight


export const getUserMessagesCount = (messages: BotMessage[]) => {
  if (!messages) {
    return 0
  }

  const keys = Object.keys(messages)

  return keys.reduce((count, key) => {
    const idx = Number(key)
    return !messages[idx].is_ai ? count + 1 : count
  }, 0)
}

const THINK_REGEX = /<think>([\s\S]*?)<\/think>/g;
const TOOLCALL_REGEX = /<toolcall>([\s\S]*?)<\/toolcall>/g;

export function unescapeHtml(escaped: string): string {
  return escaped
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'");
}

const THINK_OPEN = '<think>';
const THINK_CLOSE = '</think>';

/* WIP: Rendering refactoring must be done in the future */
function transformThinkingBlocksPartial(text: string): string {
  let result = '';
  let currentIndex = 0;

  while (true) {
    const openIdx = text.indexOf(THINK_OPEN, currentIndex);
    if (openIdx === -1) {
      result += text.slice(currentIndex);
      break;
    }

    result += text.slice(currentIndex, openIdx);

    const afterOpen = openIdx + THINK_OPEN.length;
    const closeIdx = text.indexOf(THINK_CLOSE, afterOpen);
    if (closeIdx === -1) {
      const partialContent = text.slice(afterOpen);
      result += makeThinkblockHTML(partialContent, false);
      break;
    } else {
      const finalContent = text.slice(afterOpen, closeIdx);
      result += makeThinkblockHTML(finalContent, true);
      currentIndex = closeIdx + THINK_CLOSE.length;
    }
  }

  return result;
}

function transformThinkingBlocksFinal(text: string): string {
  return text.replace(THINK_REGEX, (_, innerContent) => {
    return makeThinkblockHTML(innerContent, true);
  });
}

function makeThinkblockHTML(content: string, isFinal: boolean): string {
  const status = isFinal ? 'final' : 'partial';
  let json = JSON.stringify(content);
  json = json
    .replace(/'/g, '\\u0027')
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e')
    .replace(/&/g, '\\u0026');

  return `

<thinkblock data-think='${json}' data-status='${status}'></thinkblock>

`;
}

function makeToolCallHTML(content: string): string {
  let json = JSON.stringify(content);

  json = json
    .replace(/'/g, '\\u0027')
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e')
    .replace(/&/g, '\\u0026');

  return `

<toolcall data-json='${json}'></toolcall>

`;
}

function transformToolCallBlocksFinal(text: string): string {
  return text.replace(TOOLCALL_REGEX, (_, innerContent: string) => {
    return makeToolCallHTML(innerContent);
  });
}

export function transformAllCustomTags(text: string): string {
  let result = text;

  if (text.includes("<think>") && text.includes("</think>")) {
    result = transformThinkingBlocksFinal(text);
  }

  if (result.includes("<toolcall>") && result.includes("</toolcall>")) {
    result = transformToolCallBlocksFinal(result);
  }

  return result;
}