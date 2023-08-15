/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */
import { ReducerAction } from 'react'

import { availableTags } from 'components/DbLabInstanceForm/utils'

export const initialState = {
  isLoading: false,
  formStep: 'create',
  api_name: 'ssd',
  name: '',
  publicKeys: '',
  tag: availableTags[0],
  verificationToken: '',
}

export const reducer = (
  state: typeof initialState,
  // @ts-ignore
  action: ReducerAction<unknown, void>,
) => {
  switch (action.type) {
    case 'change_name': {
      return {
        ...state,
        name: action.name,
      }
    }
    case 'change_verification_token': {
      return {
        ...state,
        verificationToken: action.verificationToken,
      }
    }
    case 'change_public_keys': {
      return {
        ...state,
        publicKeys: action.publicKeys,
      }
    }
    case 'set_form_step': {
      return {
        ...state,
        formStep: action.formStep,
      }
    }
    case 'set_tag': {
      return {
        ...state,
        tag: action.tag,
      }
    }
    default:
      throw new Error()
  }
}
