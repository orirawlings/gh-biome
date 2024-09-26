# gh-biome

A GitHub (`gh`) CLI extension to store many git repos fetched from independent remotes in a single local git repo, a.k.a. a "git biome". By storing all git objects and references from many repos in a common database, we enable fast bulk analysis and querying across all repos.

This tool helps manage the initialization, configuration, and maintenance of the local git biome repo.


## Installation

1. Install the `gh` CLI - see the [installation](https://github.com/cli/cli#installation)

   _Installation requires a minimum version (2.0.0) of the GitHub CLI that supports extensions._

2. Install this extension:

   ```sh
   gh extension install orirawlings/gh-biome
   ```

<details>
   <summary>Installing Manually</summary>

> If you want to install this extension **manually**, follow these steps:

1. Clone the repo

   ```shell
   # git
   git clone https://github.com/orirawlings/gh-biome
   ```

   ```shell
   # GitHub CLI
   gh repo clone orirawlings/gh-biome
   ```

2. Cd into it

   ```bash
   cd gh-biome
   ```

3. Build it

   ```bash
   go build
   ```

4. Install it locally
   ```bash
   gh extension install .
   ```
   </details>
