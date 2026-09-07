<!--
  Spec template. One delta operation per `## ` heading. Worked example:

      ## ADDED Requirements

      ### Requirement: User can export data
      The system SHALL allow users to export their data in CSV format.

      #### Scenario: Successful export
      - **WHEN** user clicks "Export" button
      - **THEN** system downloads a CSV file with all user data

      ## REMOVED Requirements

      ### Requirement: Legacy export
      **Reason**: Replaced by new export system.
      **Migration**: Use new export endpoint at /v2/export.

  Critical: scenario headers use exactly four hashes (####). Three hashes or
  bullets fail the apply parser silently.
-->

## ADDED Requirements

### Requirement: <!-- requirement name -->
<!-- requirement text -->

#### Scenario: <!-- scenario name -->
- **WHEN** <!-- condition -->
- **THEN** <!-- expected outcome -->
