export type ConfigMode = 'simple' | 'expert'

// Picks the default Configuration tab when the page first finishes loading
// the server config. An unconfigured instance lands on Simple; an instance
// that already has a source host filled in OR a non-logical retrieval mode
// configured lands on Expert so the user sees the form they previously
// interacted with. Physical-mode users do not have a host field but still
// need to land on Expert because Simple-mode targets logical retrieval.
export const getInitialConfigMode = (
  host: string | undefined,
  retrievalMode?: string,
): ConfigMode => (host || retrievalMode === 'physical' ? 'expert' : 'simple')
