Feature: User Authentication
  As a user
  I want to log in securely
  So that I can access my nutrition data

  Background:
    Given the API is running
    And the database is clean

  Scenario: Successful login with valid credentials
    Given I am a registered user with email "user@example.com"
    When I submit login credentials:
      | email              | password    |
      | user@example.com   | SecurePass1 |
    Then I should receive a JWT token
    And the token should be valid for 24 hours
    And my login should be logged with trace ID
    And no PII should be in the logs

  Scenario: Failed login with invalid password
    Given I am a registered user with email "user@example.com"
    When I submit login credentials:
      | email              | password    |
      | user@example.com   | WrongPass   |
    Then I should receive a 401 Unauthorized response
    And the error message should be "Invalid credentials"
    And the failed attempt should be logged
    And my account should not be locked after 1 attempt

  Scenario: Account lockout after multiple failed attempts
    Given I am a registered user with email "user@example.com"
    When I submit invalid credentials 5 times
    Then my account should be locked
    And I should receive a 423 Locked response
    And an alert should be triggered
    And the lockout should be logged

  Scenario: Password reset request
    Given I am a registered user with email "user@example.com"
    When I request a password reset
    Then I should receive a reset token via email
    And the token should expire in 1 hour
    And the request should be logged
    And the token should be hashed in the database

  Scenario: JWT token refresh
    Given I have a valid JWT token
    When the token is about to expire in 5 minutes
    And I request a token refresh
    Then I should receive a new JWT token
    And the old token should be invalidated
    And the refresh should be logged

  Scenario: Logout
    Given I am logged in with a valid token
    When I logout
    Then my token should be invalidated
    And I should receive a 200 OK response
    And the logout should be logged
