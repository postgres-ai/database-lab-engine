/* eslint-disable no-undef */

Cypress.on('uncaught:exception', () => {
  return false
})

// Function to set up intercepts for the requests
function setupIntercepts() {
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
}

describe('Configuration tab', () => {
  beforeEach(() => {
    cy.visit('/')
    setupIntercepts()
  })

  it('should have "Configuration" tab with form inputs', () => {
    cy.get('.MuiTabs-flexContainer').contains('Configuration').click({
      force: true,
    })

    cy.get('input[type="text"]').should('exist')
    cy.get('input[type="checkbox"]').should('exist')
    cy.get('button[type="button"]').should('exist')

    cy.get('button').contains('Cancel').click({
      force: true,
    })
    cy.url().should('eq', Cypress.config().baseUrl + '/instance')
  })
})
