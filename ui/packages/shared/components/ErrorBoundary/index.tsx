import React, { Component, ErrorInfo, ReactNode } from 'react'

import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'

type Props = {
  children: ReactNode
  // Rendered in place of the default stub when the subtree throws.
  fallback?: ReactNode
}

type State = {
  error: Error | null
}

// React unmounts the entire tree when a render throws and no boundary catches it, leaving a
// blank page with nothing pointing at the cause.
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Unhandled UI error:', error, info.componentStack)
  }

  render() {
    if (!this.state.error) return this.props.children

    if (this.props.fallback) return this.props.fallback

    return (
      <ErrorStub
        title="Something went wrong"
        message={this.state.error.message}
      />
    )
  }
}
