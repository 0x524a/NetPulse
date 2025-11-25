name: Feature Request
description: Suggest a new feature or enhancement
title: "[FEATURE] "
labels: ["enhancement"]
assignees: []

body:
  - type: markdown
    attributes:
      value: |
        Thank you for suggesting a feature! Please provide as much detail as possible.

  - type: input
    id: title
    attributes:
      label: Feature Title
      description: Brief title of the feature
      placeholder: "e.g., Add JSON output format"
    validations:
      required: true

  - type: textarea
    id: description
    attributes:
      label: Description
      description: What feature would you like to see?
      placeholder: "Clear description of the feature request"
    validations:
      required: true

  - type: textarea
    id: motivation
    attributes:
      label: Motivation & Use Case
      description: Why do you need this feature?
      placeholder: "What problem does this solve or what benefit does it provide?"
    validations:
      required: true

  - type: textarea
    id: solution
    attributes:
      label: Proposed Solution
      description: How should this feature work?
      placeholder: "Describe how you imagine using this feature"

  - type: textarea
    id: alternatives
    attributes:
      label: Alternatives Considered
      description: Have you considered other approaches?
      placeholder: "Any alternative solutions or workarounds?"

  - type: textarea
    id: additional
    attributes:
      label: Additional Context
      description: Any other context or examples
