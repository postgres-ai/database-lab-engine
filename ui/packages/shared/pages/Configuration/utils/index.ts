import { DatabaseType } from "types/api/entities/config"

export const uniqueDatabases = (values: string) => {
  let splitValuesArray = values.split(/[,(\s)(\n)(\r)(\t)(\r\n)]/)
  let newValuesArray = []

  for (let i in splitValuesArray) {
    if (
      splitValuesArray[i] !== '' &&
      newValuesArray.indexOf(splitValuesArray[i]) === -1
    ) {
      newValuesArray.push(splitValuesArray[i])
    }
  }

  return newValuesArray.join(',')
}

export const postUniqueDatabases = (values: any) => {
  let splitValuesArray = values.split(/[,(\s)(\n)(\r)(\t)(\r\n)]/)

  const obj = splitValuesArray.reduce((acc: DatabaseType, curr: number) => {
    acc[curr] = {}
    return acc
  }, {})

  return values.length !== 0 ? obj : null
}
