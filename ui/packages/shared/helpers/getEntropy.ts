const replaceChars = '!@$&*'
const sepChars = '_-., '
const otherSpecialChars = '“#%"()+/:;<=>?[\\]^{|}~'
const lowerChars = 'abcdefghijklmnopqrstuvwxyz'
const upperChars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
const digitsChars = '0123456789'
export const MIN_ENTROPY = 60

function getBase(password: string): number {
  let uniqueChars: string[] = []
  for (const c of password) {
    if (!uniqueChars.includes(c)) {
      uniqueChars.push(c)
    }
  }
  let hasReplace = false
  let hasSep = false
  let hasOtherSpecial = false
  let hasLower = false
  let hasUpper = false
  let hasDigits = false
  let base = 0

  for (let i = 0; i < uniqueChars.length; i++) {
    switch (true) {
      case replaceChars.includes(uniqueChars[i]):
        hasReplace = true
        break
      case sepChars.includes(uniqueChars[i]):
        hasSep = true
        break
      case otherSpecialChars.includes(uniqueChars[i]):
        hasOtherSpecial = true
        break
      case lowerChars.includes(uniqueChars[i]):
        hasLower = true
        break
      case upperChars.includes(uniqueChars[i]):
        hasUpper = true
        break
      case digitsChars.includes(uniqueChars[i]):
        hasDigits = true
        break
      default:
        base++
        break
    }
  }
  if (hasReplace) {
    base += replaceChars.length
  }
  if (hasSep) {
    base += sepChars.length
  }
  if (hasOtherSpecial) {
    base += otherSpecialChars.length
  }
  if (hasLower) {
    base += lowerChars.length
  }
  if (hasUpper) {
    base += upperChars.length
  }
  if (hasDigits) {
    base += digitsChars.length
  }
  return base
}
const seqNums = '0123456789'
const seqKeyboard0 = 'qwertyuiop'
const seqKeyboard1 = 'asdfghjkl'
const seqKeyboard2 = 'zxcvbnm'
const seqAlphabet = 'abcdefghijklmnopqrstuvwxyz'
function removeMoreThanTwoFromSequence(s: string, seq: string): string {
  const seqRunes: string[] = Array.from(seq)
  let runes: string[] = Array.from(s)
  let matches = 0
  for (let i = 0; i < runes.length; i++) {
    for (let j = 0; j < seqRunes.length; j++) {
      if (i >= runes.length) {
        break
      }
      const r = runes[i]
      const r2 = seqRunes[j]
      if (r !== r2) {
        matches = 0
        continue
      }
      // found a match, advance the counter
      matches++
      if (matches > 2) {
        runes.splice(i, 1)
      } else {
        i++
      }
    }
  }
  return runes.join('')
}
function getReversedString(s: string): string {
  const rune: string[] = Array.from(s)
  const n = rune.length
  for (let i = 0; i < Math.floor(n / 2); i++) {
    ;[rune[i], rune[n - 1 - i]] = [rune[n - 1 - i], rune[i]]
  }
  return rune.join('')
}
function removeMoreThanTwoRepeatingChars(s: string): string {
  let prevPrev: string = ''
  let prev: string = ''
  const runes: string[] = Array.from(s)
  for (let i = 0; i < runes.length; i++) {
    const r = runes[i]
    if (r === prev && r === prevPrev) {
      runes.splice(i, 1)
      i--
    }
    prevPrev = prev
    prev = r
  }
  return runes.join('')
}
function getLength(password: string): number {
  password = removeMoreThanTwoRepeatingChars(password)
  password = removeMoreThanTwoFromSequence(password, seqNums)
  password = removeMoreThanTwoFromSequence(password, seqKeyboard0)
  password = removeMoreThanTwoFromSequence(password, seqKeyboard1)
  password = removeMoreThanTwoFromSequence(password, seqKeyboard2)
  password = removeMoreThanTwoFromSequence(password, seqAlphabet)
  password = removeMoreThanTwoFromSequence(password, getReversedString(seqNums))
  password = removeMoreThanTwoFromSequence(
    password,
    getReversedString(seqKeyboard0),
  )
  password = removeMoreThanTwoFromSequence(
    password,
    getReversedString(seqKeyboard1),
  )
  password = removeMoreThanTwoFromSequence(
    password,
    getReversedString(seqKeyboard2),
  )
  password = removeMoreThanTwoFromSequence(
    password,
    getReversedString(seqAlphabet),
  )
  return password.length
}
export function getEntropy(password: string): number {
  return getEntropyInternal(password)
}
function getEntropyInternal(password: string): number {
  const base = getBase(password)
  const length = getLength(password)
  // calculate log2(base^length)
  return logPow(base, length, 2)
}
function logX(base: number, n: number): number {
  if (base == 0) {
    return 0
  } else {
    return Math.log2(n) / Math.log2(base)
  }
}
function logPow(expBase: number, pow: number, logBase: number): number {
  let total = 0
  for (let i = 0; i < pow; i++) {
    total += logX(logBase, expBase)
  }
  return total
}

function getSecureRandomInt(max: number): number {
  const array = new Uint32Array(1)
  crypto.getRandomValues(array)
  return array[0] % max
}

export function generatePassword(length: number = 16): string {
  const minLength = 4
  const actualLength = Math.max(length, minLength)

  const lowercase = 'abcdefghijklmnopqrstuvwxyz'
  const uppercase = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
  const digits = '0123456789'
  const special = '!@$&*_-.'
  const allChars = lowercase + uppercase + digits + special

  let password = ''
  // ensure at least one character from each category
  password += lowercase[getSecureRandomInt(lowercase.length)]
  password += uppercase[getSecureRandomInt(uppercase.length)]
  password += digits[getSecureRandomInt(digits.length)]
  password += special[getSecureRandomInt(special.length)]

  // fill the rest with random characters
  for (let i = password.length; i < actualLength; i++) {
    password += allChars[getSecureRandomInt(allChars.length)]
  }

  // shuffle the password to randomize positions (Fisher-Yates)
  const shuffled = password.split('')
  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = getSecureRandomInt(i + 1)
    ;[shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]]
  }

  return shuffled.join('')
}

export function validatePassword(password: string, minEntropy: number): string {
  const entropy: number = getEntropy(password)
  if (entropy >= minEntropy) {
    return ''
  }

  let hasReplace: boolean = false
  let hasSep: boolean = false
  let hasOtherSpecial: boolean = false
  let hasLower: boolean = false
  let hasUpper: boolean = false
  let hasDigits: boolean = false

  for (const c of password) {
    switch (true) {
      case replaceChars.includes(c):
        hasReplace = true
        break
      case sepChars.includes(c):
        hasSep = true
        break
      case otherSpecialChars.includes(c):
        hasOtherSpecial = true
        break
      case lowerChars.includes(c):
        hasLower = true
        break
      case upperChars.includes(c):
        hasUpper = true
        break
      case digitsChars.includes(c):
        hasDigits = true
        break
    }
  }

  const allMessages: string[] = []

  if (!hasOtherSpecial || !hasSep || !hasReplace) {
    allMessages.push('including more special characters')
  }
  if (!hasLower) {
    allMessages.push('using lowercase letters')
  }
  if (!hasUpper) {
    allMessages.push('using uppercase letters')
  }
  if (!hasDigits) {
    allMessages.push('using numbers')
  }

  if (allMessages.length > 0) {
    const errorMessage: string = `Weak password, try ${allMessages.join(
      ', ',
    )} or using a longer password`
    return errorMessage
  }

  return 'Weak password, try using a longer password'
}
