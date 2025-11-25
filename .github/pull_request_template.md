name: Pull Request Template
description: Standard pull request template
title: "[PR] "

body:
  - type: markdown
    attributes:
      value: |
        Thank you for contributing to NetPulse! Please fill in the following information:

  - type: input
    id: title
    attributes:
      label: PR Title
      description: Clear, descriptive title
      placeholder: "e.g., Add JSON output support for benchmarks"
    validations:
      required: true

  - type: textarea
    id: description
    attributes:
      label: Description
      description: What does this PR do?
      placeholder: "Describe the changes and why they are needed"
    validations:
      required: true

  - type: textarea
    id: type
    attributes:
      label: Type of Change
      description: What type of change is this?
      value: |
        - [ ] Bug fix (fixes an issue)
        - [ ] New feature (adds functionality)
        - [ ] Breaking change (would cause existing functionality to change)
        - [ ] Documentation update

  - type: textarea
    id: testing
    attributes:
      label: Testing
      description: How did you test these changes?
      value: |
        - [ ] Unit tests added/updated
        - [ ] Tested locally
        - [ ] All tests pass: `go test ./...`
        - [ ] No breaking changes

  - type: textarea
    id: checklist
    attributes:
      label: Checklist
      description: Make sure you've completed these
      value: |
        - [ ] Code follows Go style guidelines (go fmt, go vet)
        - [ ] All tests pass
        - [ ] New tests added for new functionality
        - [ ] Documentation updated
        - [ ] Commit messages are clear
        - [ ] No unnecessary dependencies added

  - type: textarea
    id: related
    attributes:
      label: Related Issues
      description: Link any related issues
      placeholder: "Fixes #123 or Related to #456"

  - type: textarea
    id: additional
    attributes:
      label: Additional Context
      description: Any additional information
