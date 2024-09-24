import { isMobileDevice } from "../../../utils/utils";

export const checkIsSendCmd = (e: KeyboardEvent): boolean => {
  if (isMobileDevice()) {
    return false; // On mobile devices, Enter should not send.
  }
  return e.code === 'Enter' && !e.shiftKey && !e.ctrlKey && !e.metaKey;
};

export const checkIsNewLineCmd = (e: KeyboardEvent): boolean => {
  if (isMobileDevice()) {
    return e.code === 'Enter'; // On mobile devices, Enter should create a new line.
  }
  return e.code === 'Enter' && (e.shiftKey || e.ctrlKey || e.metaKey);
};

export const addNewLine = (
  value: string,
  element: HTMLInputElement | HTMLTextAreaElement,
) => {
  const NEW_LINE_STR = '\n'

  const firstLineLength = element.selectionStart ?? value.length
  const secondLineLength = element.selectionEnd ?? value.length

  const firstLine = value.substring(0, firstLineLength)
  const secondLine = value.substring(secondLineLength)

  return {
    value: `${firstLine}${NEW_LINE_STR}${secondLine}`,
    caretPosition: firstLineLength + NEW_LINE_STR.length,
  }
}

export const checkIsPrevMessageCmd = (
  e: KeyboardEvent,
  element: HTMLInputElement | HTMLTextAreaElement,
) => {
  const isRightKey =
    e.code === 'ArrowUp' && !e.ctrlKey && !e.metaKey && !e.shiftKey

  // Use prev message only if the caret is in the start of the input.
  const targetCaretPosition = 0

  const isRightCaretPosition =
    element.selectionStart === targetCaretPosition &&
    element.selectionEnd === targetCaretPosition

  return isRightKey && isRightCaretPosition
}

export const checkIsNextMessageCmd = (
  e: KeyboardEvent,
  element: HTMLInputElement | HTMLTextAreaElement,
) => {
  const isRightKey =
    e.code === 'ArrowDown' && !e.ctrlKey && !e.metaKey && !e.shiftKey

  // Use next message only if the caret is in the end of the input.
  const targetCaretPosition = element.value.length

  const isRightCaretPosition =
    element.selectionStart === targetCaretPosition &&
    element.selectionEnd === targetCaretPosition

  return isRightKey && isRightCaretPosition
}
