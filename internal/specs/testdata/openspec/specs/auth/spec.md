# Auth Specification

## Purpose
Authentication and session management for the application.

## Requirements

### Requirement: User Authentication
The system MUST issue a session after successful login.

#### Scenario: Valid credentials
- GIVEN a user with valid credentials
- WHEN the user signs in
- THEN a session is created
