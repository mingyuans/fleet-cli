## ADDED Requirements

### Requirement: User can specify PR target branch via --base flag
The `fleet pr` command SHALL accept a `--base` (short: `-b`) flag that specifies the target branch for the PR. When `--base` is not provided, the command SHALL use the manifest `revision` as the target branch (existing behavior).

#### Scenario: Single target branch specified and exists
- **WHEN** user runs `fleet pr --base testing` and the `testing` branch exists on the fetch remote
- **THEN** the PR SHALL be created with `testing` as the base branch

#### Scenario: Single target branch specified but does not exist
- **WHEN** user runs `fleet pr --base nonexist` and the `nonexist` branch does not exist on the fetch remote
- **THEN** the PR creation for that project SHALL be skipped with a message indicating no matching base branch was found

#### Scenario: No --base flag provided
- **WHEN** user runs `fleet pr` without the `--base` flag
- **THEN** the command SHALL use the manifest `revision` (with `masterMainCompat` logic) as the base branch, maintaining existing behavior

### Requirement: Pipe-separated fallback branch candidates
The `--base` flag SHALL support `|` as a separator for multiple candidate branches. The system SHALL try each candidate from left to right and use the first branch that exists on the fetch remote.

#### Scenario: First candidate does not exist, second exists
- **WHEN** user runs `fleet pr --base "testing-incy|testing"` and `testing-incy` does not exist but `testing` exists on the fetch remote
- **THEN** the PR SHALL be created with `testing` as the base branch

#### Scenario: All candidates do not exist
- **WHEN** user runs `fleet pr --base "nonexist1|nonexist2"` and neither branch exists on the fetch remote
- **THEN** the PR creation for that project SHALL be skipped with a clear error message listing the attempted candidates

#### Scenario: Candidates with whitespace and empty segments
- **WHEN** user runs `fleet pr --base " testing | "` (with extra spaces and empty segments)
- **THEN** the system SHALL trim whitespace and ignore empty segments, treating it as a single candidate `testing`

### Requirement: Fetch remote before branch existence check
When `--base` is specified, the system SHALL fetch the fetch remote before checking branch existence to ensure the local refs are up to date.

#### Scenario: Branch recently created on remote
- **WHEN** user runs `fleet pr --base testing` and `testing` was recently created on the remote but not yet fetched locally
- **THEN** the system SHALL fetch the remote first and correctly detect the branch existence

### Requirement: No masterMainCompat when --base is specified
When `--base` is explicitly specified, the `masterMainCompat` fallback logic SHALL NOT be applied. The `|` operator provides the user's own fallback mechanism.

#### Scenario: --base master without compat
- **WHEN** user runs `fleet pr --base master` and only `main` exists on the remote (no `master`) with `masterMainCompat` enabled
- **THEN** the PR creation SHALL be skipped (NOT fall back to `main`), because the user explicitly specified `master`
