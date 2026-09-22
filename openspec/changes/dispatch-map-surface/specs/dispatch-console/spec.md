## ADDED Requirements

### Requirement: The console links to the map

The console SHALL offer a way to reach the map, and the map SHALL offer a way back. The
link SHALL be composed from `@mssfoobar/ui` primitives. This change SHALL NOT introduce a
sidebar or a navigation component — the baseline deliberately has neither, and one link in
each direction is enough for two pages.

#### Scenario: Reaching the map from the console
- **WHEN** an operator is on the console
- **THEN** a **Map** link is visible and navigates to the map without a full page load

#### Scenario: Returning to the console from the map
- **WHEN** an operator is on the map
- **THEN** a link back to the console is visible and navigates to it

#### Scenario: No navigation component is added
- **WHEN** the app's source is reviewed
- **THEN** it contains no `nav.ts`, no `Sidebar` and no `Headerbar`, exactly as before
