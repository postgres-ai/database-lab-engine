import { useContext, createContext } from 'react'

export const createStrictContext = <Value>() => {
  const Context = createContext<Value>(undefined as unknown as Value)

  return {
    useStrictContext: () => useContext(Context),
    Provider: Context.Provider,
  }
}
