/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { LocalStorage as LocalStorageShared } from '@postgres.ai/shared/helpers/localStorage'

class LocalStorage extends LocalStorageShared {}

export const localStorage = new LocalStorage()
