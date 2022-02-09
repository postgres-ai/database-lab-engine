import { LocalStorage as LocalStorageShared } from '@postgres.ai/shared/helpers/localStorage'

class LocalStorage extends LocalStorageShared {}

export const localStorage = new LocalStorage()
