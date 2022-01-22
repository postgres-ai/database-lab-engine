First off, thanks for taking the time to contribute! 💪🐘🎉

The following is a set of guidelines for contributing to Database Lab Engine (DLE) and its additional components, which are hosted on GitLab and GitHub:
- https://gitlab.com/postgres-ai/database-lab
- https://github.com/postgres-ai/database-lab-engine

These are mostly guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

---

#### Table of contents

- [Code of conduct](#code-of-conduct)
- [TL;DR – I just have a question, where to ask it?](#tldr-i-just-have-a-question-where-to-ask-it)
- [How can I contribute?](#how-can-i-contribute)
    - [Reporting bugs](#reporting-bugs)
    - [Proposing enhancements](#proposing-enhancements)
    - [Your first code contribution](#your-first-code-contribution)
    - [Roadmap](#roadmap)
    - [Merge Requests / Pull Requests](#merge-requests-pull-requests)
- [Development guides](#repo-overview)
    - [Git commit messages](#git-commit-messages)
    - [Go styleguide](#go-styleguide)
    - [Documentation styleguide](#documentation-styleguide)
    - [Development setup](#development-setup)
<!--
    - [Repo overview](#repo-overview)
    - [Building from source](#building-from-source)
-->

---

## Code of conduct
This project and everyone participating in it are governed by the [Database Lab Engine Community Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [community@postgres.ai](mailto:community@postgres.ai).

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](./CODE_OF_CONDUCT.md)

## TL;DR – I just have a question, where to ask it?

> **Note:** Please don't open an issue to just ask a question. You will get faster results by using the channels below.

- Fast ways to get in touch: [Contact page](https://postgres.ai/contact)
- [Database Lab Community Slack](https://slack.postgres.ai) (English)
- [Telegram](https://t.me/databaselabru) (Russian)

## How can I contribute?
### Reporting bugs
- Use a clear and descriptive title for the issue to identify the problem.
- Make sure you test against the latest released version. We may have already fixed the bug you're experiencing.
- Provide steps to reproduce the issue, including DLE version, PostgreSQL version, and the platform you are running on (some examples: RDS, self-managed Postgres on an EC2 instance, self-managed Postgres on-prem).
- Explain which behavior you expected to see instead and why.
- Please include DLE logs. Include Postgres logs from clones and/or the sync container's Postgres, if relevant.
- Describe DLE configuration: mode you are using (physical or logical), other details of DLE configuration, Postgres configuration snippets.
- If the issue is related to UI, include screenshots and animated GIFs. Please, do NOT use screenshots for console output, configs, and logs – for those, always prefer the textual form.
- Check if you have sensitive information in the logs and configs and remove any.
- You can submit a bug report in either [GitLab Issues](https://gitlab.com/postgres-ai/database-lab) or [GitHub Issues](https://github.com/postgres-ai/database-lab-engine) sections – both places are monitored.
- If you believe that there is an urgency related to the reported bug, feel free to reach out to the project maintainers. Additionally, using one of [the channels described above](#tldr-i-just-have-a-question-where-to-ask-it).
- If you need to report a security issue, follow instructions in ["Database Lab Engine security guidelines"](./SECURITY.md).

### Proposing enhancements
This section guides you through submitting an enhancement suggestion for DLE, including completely new features and minor improvements to existing functionality. Following these guidelines helps maintainers and the community understand your suggestion and find related proposals.

When you are creating an enhancement suggestion, please include as many details as possible. Include the steps that you imagine you would take if the feature you're requesting existed.

Enhancement suggestions are tracked on [GitLab](https://gitlab.com/postgres-ai/database-lab) or [GitHub](https://github.com/postgres-ai/database-lab-engine). Recommendations:

- Use a clear and descriptive title for the issue to identify the suggestion.
- Provide a step-by-step description of the proposed enhancement in as many details as possible.
- Provide specific examples to demonstrate the steps. Include copy/pasteable snippets which you use in those examples
- Use Markdown to format code snippets and improve the overall look of your issues (Markdown docs: [GitLab](https://docs.gitlab.com/ee/user/markdown.html), [GitHub](https://github.github.com/gfm/)).
- Describe the current behavior and explain which behavior you expected to see instead and why.
- If your proposal is related to UI, include screenshots and animated GIFs which help you demonstrate the steps or point out the part of DLE to which the suggestion is related. Please, do NOT use screenshots for console output, logs, configs.
- Explain why this proposal would be helpful to most DLE users.
- Specify which version of DLE you're using. If it makes sense, specify Postgres versions too.
- Specify the name and version of the OS you're using.

### Your first code contribution
We appreciate first-time contributors, and we are happy to assist you in getting started. In case of any questions, reach out to us!

You find some issues that are considered as good for first-time contributors looking at [the issues with the `good-first-issue` label](https://gitlab.com/postgres-ai/database-lab/-/issues?label_name%5B%5D=good+first+issue). Assign yourself to an issue and start discussing a possible solution. It is always a good idea to discuss and collaborate before you propose an MR/PR.

### Roadmap
There is [the Roadmap section](https://postgres.ai/docs/roadmap) in the docs. It contains some major items defining big goals for the development team and the DLE community. However, these items cannot be considered a strict plan, so there are no guarantees that everything from the list will be necessarily implemented.

### Merge Requests / Pull Requests
DLE is developed on GitLab, so MRs (merge requests) there is a way to propose a contribution. GitHub PRs (pull requests) are also an option but note that eventually, the proposal will need to be moved to GitLab, so the processing time may be increased.

Please follow these steps to have your contribution considered by the maintainers:
1. Follow the [styleguides](#styleguides).
2. Provide a detailed description of your MR/PR following the same rules as you would use for opening an issue (see [Reporting Bugs](#reporting-bugs) and [Proposing Enhancements](#proposing-enhancements) above).
3. To get your MR/PR merged, you will need to sign Postgres.ai Database Lab Engine Contributor Agreement and ensure that the Postgres.ai team has received it. The template can be found here: [DLE-CA](https://bit.ly/dle-ca). Download it, fill out the fields, sign, and send to contribute@postgres.ai.

While the prerequisites above must be satisfied before having your MR/PR reviewed, the reviewer(s) may ask you to complete additional design work, tests, or other changes before your MR/PR can be ultimately accepted.

Additional materials that are worth checking out:
- [Git-related guidelines in the PostgreSQL project](https://wiki.postgresql.org/wiki/Working_with_Git)
- [GitLab flow](https://docs.gitlab.com/ee/topics/gitlab_flow.html)

## Styleguides
### Git commit messages
- Think about other people: how likely will they understand you?
- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Create clone..." not "Creates clone...")
- Limit the first line to 72 characters or less
- Reference issues and MRs/PRs liberally after the first line
- Below the first line, add a detailed description of the commit. The more details the commit message has, the better. PostgreSQL project has a good culture of writing very informative commit messages – [check out a few of them](https://git.postgresql.org/gitweb/?p=postgresql.git;a=summary) to get inspired.
- Read this: ["How to write a good commit message"](https://docs.gitlab.com/ee/topics/gitlab_flow.html#how-to-write-a-good-commit-message)
- All Git commits must be signed. Unsigned commits are rejected.
    - [How to sign a commit](https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work)
    - [How to sign what already committed (but not yet pushed)](https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work)
    - [GitLab-specific guidelines](https://docs.gitlab.com/ee/user/project/repository/gpg_signed_commits/)

### Go styleguide
We encourage you to follow the principles described in the following documents:
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Documentation styleguide
Documentation for Database Lab Engine and additional components is hosted at https://postgres.ai/docs and is maintained in this GitLab repo: https://gitlab.com/postgres-ai/docs.

We're building documentation following the principles described at https://documentation.divio.com/:

> There is a secret that needs to be understood in order to write good software documentation: there isn’t one thing called documentation, there are four.
> 
> They are: tutorials, how-to guides, technical reference and explanation. They represent four different purposes or functions, and require four different approaches to their creation. Understanding the implications of this will help improve most documentation - often immensely.

Learn more: https://documentation.divio.com/.

<!--
### Repo overview
TBD
-->

### Development setup
- Install Docker. Example for Linux:
    ```bash
    # Install dependencies
    sudo apt-get update && sudo apt-get install -y \
      apt-transport-https \
      ca-certificates \
      curl \
      gnupg-agent \
      software-properties-common
    # Install Docker
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

    sudo add-apt-repository \
      "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) \
      stable"

    sudo apt-get update && sudo apt-get install -y \
      docker-ce \
      docker-ce-cli \
      containerd.io
    ```
- Install ZFS and create zpool. Example for Linux:
    ```bash
    # Install ZFS
    sudo apt-get install -y zfsutils-linux

    sudo zpool create -f \
      -O compression=on \
      -O atime=off \
      -O recordsize=128k \
      -O logbias=throughput \
      -m /var/lib/dblab/dblab_pool \
      dblab_pool \
      "/dev/nvme1n1" # ! check your device or use an empty file here;
                     # empty file creation example: truncate --size 10GB ./pseudo-disk-for-zfs
    ```
- Install `golangci-lint`: https://github.com/golangci/golangci-lint#install

<!-- TODO: Linux specific requirements? MacOS specific? -->

<!--
### Building from source
TBD – separately for Linux and MacOS


mention dev images: See our [GitLab Container Registry](https://gitlab.com/postgres-ai/database-lab/container_registry) to find the images built for development branches.
-->