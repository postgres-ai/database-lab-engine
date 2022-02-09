/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Dimensions } from './types';

export const checkIsVisibleLeftCurtain = (dimensions: Dimensions) => {
  return dimensions.scrollLeft > 0;
};

export const checkIsVisibleRightCurtain = (dimensions: Dimensions) => {
  return dimensions.scrollLeft < dimensions.scrollWidth - dimensions.offsetWidth;
};
