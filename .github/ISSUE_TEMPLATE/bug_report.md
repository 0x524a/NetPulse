name: Bug Report
description: Report a bug or issue
title: "[BUG] "
labels: ["bug"]
assignees: []

body:
  - type: markdown
    attributes:
      value: |
        Thank you for taking the time to report a bug! Please fill in as much detail as possible to help us investigate.

  - type: input
    id: title
    attributes:
      label: Title
      description: Brief title of the bug
      placeholder: "e.g., NetPulse crashes when testing with mlab provider"
    validations:
      required: true

  - type: textarea
    id: description
    attributes:
      label: Description
      description: Clear description of the bug
      placeholder: "What is happening that shouldn't be?"
    validations:
      required: true

  - type: textarea
    id: steps
    attributes:
      label: Steps to Reproduce
      description: Steps to reproduce the issue
      value: |
        1. Run command: 
        2. Then:
        3. Bug occurs:
    validations:
      required: true

  - type: textarea
    id: expected
    attributes:
      label: Expected Behavior
      description: What should happen instead
    validations:
      required: true

  - type: textarea
    id: actual
    attributes:
      label: Actual Behavior
      description: What actually happens
    validations:
      required: true

  - type: input
    id: go-version
    attributes:
      label: Go Version
      description: Output of `go version`
      placeholder: "go version go1.24.0 darwin/amd64"
    validations:
      required: true

  - type: input
    id: os
    attributes:
      label: Operating System
      description: What OS are you using?
      placeholder: "macOS 14.2, Ubuntu 22.04, Windows 11, etc."
    validations:
      required: true

  - type: input
    id: netpulse-version
    attributes:
      label: NetPulse Version
      description: What version of NetPulse are you using?
      placeholder: "v1.0.0, or latest from main"
    validations:
      required: true

  - type: textarea
    id: logs
    attributes:
      label: Logs/Output
      description: Relevant error messages or log output
      render: bash

  - type: textarea
    id: additional
    attributes:
      label: Additional Context
      description: Any other context that might help
