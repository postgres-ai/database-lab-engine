import { useEffect } from 'react';

const INTERCOM_BUTTON_SELECTOR = `div[aria-label="Open Intercom Messenger"]`;
const INTERCOM_IFRAME_SELECTOR = `iframe[name="intercom-launcher-frame"]`;

export const useHideIntercom = () => {
  useEffect(() => {
    const intercomButton = document.querySelector(INTERCOM_BUTTON_SELECTOR);
    const intercomIframe = document.querySelector(INTERCOM_IFRAME_SELECTOR);

    const originalButtonDisplay = intercomButton ? (intercomButton as HTMLElement).style.display : '';
    const originalIframeDisplay = intercomIframe ? (intercomIframe as HTMLElement).style.display : '';

    const hideIntercom = () => {
      if (intercomButton) {
        (intercomButton as HTMLElement).style.display = 'none';
      }
      if (intercomIframe) {
        (intercomIframe as HTMLElement).style.display = 'none';
      }
    };

    const showIntercom = () => {
      if (intercomButton) {
        (intercomButton as HTMLElement).style.display = originalButtonDisplay || 'block';
      }
      if (intercomIframe) {
        (intercomIframe as HTMLElement).style.display = originalIframeDisplay || 'inline';
      }
    };

    hideIntercom();

    return () => {
      showIntercom();
    };
  }, []);
};