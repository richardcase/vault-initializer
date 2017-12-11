## Contributing

Thank you for considering contributing to Vault Initializer. The more people that contribute the better it will be. :tada::+1:

The following is a guide for contributing and as such shouldn't be treated as hard & fast rules. Use your judgement and feel free to propose changes to this document via pull request.

## Code of Conduct

This project and everyone participating in it is governed by the [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. 

## How Can I Contribute?

### Reporting Bugs

* **Ensure the bug was not already reported** by [searching all issues](https://github.com/richardcase/vault-initializer/issues?utf8=%E2%9C%93&q=is%3Aissue).

* If you're unable to find an open issue addressing the problem,
  [open a new one][new issue]. Be sure to include a **title and clear
  description**, as much relevant information as possible, and a **code sample**
  or an **executable test case** demonstrating the expected behavior that is not
  occurring.

### Suggesting Enhancements

We welcome new suggestions for enhancements. Enhancement suggestions are tracked as [GitHub issues](https://guides.github.com/features/issues/). To suggest a new enhancement do the following create an issue and provide the following:

* **Use a clear and descriptive title** for the issue to identify the suggestion.
* **Provide a step-by-step description of the suggested enhancement** in as many details as possible.
* **Provide specific examples to demonstrate the steps**. Include copy/pasteable snippets which you use in those examples, as [Markdown code blocks](https://help.github.com/articles/markdown-basics/#multiple-lines).
* **Describe the current behavior** and **explain which behavior you expected to see instead** and why.
* **Explain why this enhancement would be useful**
* **Specify which version of Vault Initializer you're using.** 
* **Specify the name and version of the OS you're using.**

### Your First Code Contribution

Look through the open issues and implement your own new functionality.

#### Local Development

Vault Initializer can be developed locally with Kubernetes & Vault running on your developer workstation. Additionally, Vault Initializer can run outside of Kubernetes to aid debugging and to speed up the development lifecycle.

The steps to develop locally are included [here](docs/local_development.md).

### Pull Requests

* Do not include issue numbers in the PR title
* Ensure you have run the tests & linter (make ci)
* End all files with a newline
* Avoid platform-dependent code

## Styleguides

### Git Commit Messages

* Use the present tense ("Add feature" not "Added feature")
* Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
* Limit the first line to 72 characters or less
* Reference issues and pull requests liberally after the first line
