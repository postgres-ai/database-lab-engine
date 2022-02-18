const keys = {
  // Keep this name.
  authToken: 'token',
}

export class LocalStorage {
  // Auth token.
  getAuthToken = () => window.localStorage.getItem(keys.authToken)

  setAuthToken = (value: string) => window.localStorage.setItem(keys.authToken, value)

  removeAuthToken = () => window.localStorage.removeItem(keys.authToken)
}

export const localStorage = new LocalStorage()
