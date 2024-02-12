/* eslint-disable no-undef */

Cypress.on('uncaught:exception', () => {
  return false
})

// Function to set up intercepts for the requests
function setupIntercepts() {
  const exceptions = [
    '/healthz',
    '/instance/retrieval',
    '/status',
    '/admin/config',
  ]

  cy.intercept('GET', '/healthz*', {
    statusCode: 200,
    body: {
      edition: 'standard',
    },
  })

  cy.intercept('GET', '/instance/retrieval*', {
    statusCode: 200,
    body: {
      status: 'inactive',
    },
  })

  cy.intercept('GET', '/status*', {
    statusCode: 200,
    body: {
      status: {
        code: 'OK',
        message: 'Instance is ready',
      },
      pools: [],
      cloning: {
        clones: [],
      },
      retrieving: {
        status: 'inactive',
      },
    },
  })

  // Intercept all fetch requests and return a 200
  cy.intercept('GET', '*', (req) => {
    if (
      req.resourceType === 'fetch' &&
      exceptions.every((e) => !req.url.includes(e))
    ) {
      req.reply({
        statusCode: 200,
        body: {
          status: 'active',
        },
      })
    }
  })
}

describe('Configuration tab', () => {
  // It should intercept the requests
  beforeEach(() => {
    setupIntercepts()
  })

  it('should have a "Configuration" tab', () => {
    cy.visit('/', {
      retryOnStatusCodeFailure: true,
      onLoad: () => {
        cy.get('.MuiTabs-flexContainer')
          .contains('Configuration', { timeout: 10000 })
          .should('be.visible')
          .click({ force: true })
      },
    })
  })

  it('should have form inputs in the "Configuration" tab', () => {
    cy.get('.MuiTabs-flexContainer')
      .contains('Configuration', { timeout: 10000 })
      .should('be.visible')
      .click({ force: true })

    cy.get('button[type="button"]').should('exist')
  })
})
