/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const destroyRestriction = (cloneId: string) => {
  const message = `The clone "${cloneId}" is marked as protected. To destroy it, disable the destroy protection first.`
  window.alert(message)
}

export const getResetApprove = (cloneId: string) => {
  const message = `Are you sure you want to reset the Database Lab clone: "${cloneId}"?`
  return window.confirm(message)
}

export const getDestroyApprove = (cloneId: string) => {
  const message = `Are you sure you want to destroy the Database Lab clone: "${cloneId}"?`
  return window.confirm(message)
}
