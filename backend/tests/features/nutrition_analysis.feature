Feature: Nutrition Analysis
  As a user
  I want to analyze food nutrition
  So that I can track my dietary intake

  Background:
    Given I am authenticated
    And the nutrition database is available

  Scenario: Analyze a common food item
    When I request nutrition analysis for:
      | food   | quantity | unit |
      | apple  | 100      | g    |
    Then I should receive nutritional information:
      | calories | protein | carbs | fat  | fiber |
      | 52       | 0.3     | 14    | 0.2  | 2.4   |
    And the response time should be under 200ms
    And the request should be logged with trace ID

  Scenario: Analyze with halal verification
    When I request nutrition analysis with halal check for:
      | food    | quantity | unit | checkHalal |
      | chicken | 150      | g    | true       |
    Then I should receive nutritional information
    And the halal status should be "true"
    And the verification should be logged

  Scenario: Analyze non-halal food
    When I request nutrition analysis with halal check for:
      | food  | quantity | unit | checkHalal |
      | pork  | 100      | g    | true       |
    Then I should receive nutritional information
    And the halal status should be "false"
    And a warning should be included

  Scenario: Invalid food quantity
    When I request nutrition analysis for:
      | food   | quantity | unit |
      | apple  | -10      | g    |
    Then I should receive a 400 Bad Request response
    And the error should be "Quantity must be between 0 and 10000"
    And the validation error should be logged

  Scenario: Rate limiting on nutrition analysis
    Given I have made 100 requests in the last minute
    When I make another nutrition analysis request
    Then I should receive a 429 Too Many Requests response
    And the response should include "X-RateLimit-Reset" header
    And the rate limit hit should be logged

  Scenario: Nutrition analysis with medical conditions
    Given I have medical conditions: ["diabetes", "hypertension"]
    When I request nutrition analysis for:
      | food  | quantity | unit |
      | bread | 50       | g    |
    Then I should receive nutritional information
    And I should receive medical recommendations
    And the recommendations should mention "low sodium"
    And the recommendations should mention "low glycemic index"
