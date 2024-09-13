export type HintType = 'design' | 'settings' | 'performance' | 'common'

export type Hint = {
  hint: string,
  prompt: string,
  type: HintType
}

export const hints: Hint[] = [
  {
    hint: 'Help me design an IoT system DB schema',
    prompt: 'Help me design an IoT system DB schema',
    type: 'design'
  },
  {
    hint: 'Should I enable wal_compression?',
    prompt: 'Should I enable wal_compression?',
    type: 'settings',
  },
  {
    hint: 'Demonstrate benefits of Index-Only Scans',
    prompt: 'Show the benefits of Index-Only Scans. Invent a test case, create two types of queries, run them on Postgres 16, and show the plans. Then explain the difference.',
    type: 'performance',
  },
  {
    hint: 'What do people say about subtransactions?',
    prompt: 'What do people say about subtransactions?',
    type: 'common'
  },
]