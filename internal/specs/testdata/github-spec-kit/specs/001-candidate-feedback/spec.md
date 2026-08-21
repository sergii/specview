# Feature Specification: Candidate Feedback

**Feature Branch**: `001-candidate-feedback`

**Status**: Draft

## User Scenarios & Testing

### User Story 1 - Submit feedback (Priority: P1)

As an interviewer, I can submit structured feedback after an interview.

**Independent Test**: Complete an interview, submit feedback, and verify it is visible to the recruiting team.

**Acceptance Scenarios**:

1. **Given** a completed interview, **When** the interviewer submits feedback, **Then** the feedback is stored with the interviewer and timestamp.
2. **Given** an incomplete interview, **When** feedback is submitted, **Then** the request is rejected.

## Requirements

- **FR-001**: The system MUST require feedback text.
- **FR-002**: The system MUST record actor and timestamp.
