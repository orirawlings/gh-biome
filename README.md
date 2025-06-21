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

## Getting Started

For this example, we'll build a biome containing git data from some [Kubernetes](https://kubernetes.io/) related projects.

### Creating a biome

Create a new biome in a new local directory named `kubernetes`.

```
gh biome init kubernetes
```

This has created a new bare git repository in the `kubernetes/` directory. It is currently empty.

```
cd kubernetes/
git remote        # no output
git for-each-ref  # no output
```

Let's add all git repositories for the following GitHub users to the biome. This will configure a git remote for each repository owned by these owners and fetch all git references and objects from those remotes.

```
gh biome add \
   github.com/etcd-io \
   github.com/kubernetes \
   github.com/kubernetes-client \
   github.com/kubernetes-csi \
   github.com/kubernetes-sigs
```

We can list the remotes that were added.

```
git remote
```

We can list all the references that were fetched.

```
git for-each-ref
```

We can list references for just the primary branches of the remote repositories.

```
gh biome heads --all | xargs git for-each-ref
```
or
```
git for-each-ref $(gh biome heads --all)
```

Sometimes, `git for-each-ref` runs slowly after an initial fetch of all the remote repositories. We can speed it up by packing all the git references into a single file, rather than many loose ref files.

```
git maintenance run --task=pack-refs

git for-each-ref  # should run faster now
```

Sometimes object store operations can run slowly as well. We can speed it up by packing all the git objects together.

```
git maintenance run
```

We can also enable incremental maintenance on the repository.

```
git maintenance start
```

### Analyzing content

The Kubernetes communities uses [`OWNERS` files](https://www.kubernetes.dev/docs/guide/owners/) to keep track of which developers are responsible for different parts of the codebase. Our biome has downloaded all the various Kubernetes git repositories, let's analyze the `OWNERS` file content across the entire community.

To start, we can discover all the `OWNERS` files in the primary branch of a single remote repository (`github.com/kubernetes/kubernetes`). `git ls-files` will walk the source tree, outputting data for each path that matches our globbing pattern.

```
git ls-files \
   --with-tree=refs/remotes/github.com/kubernetes/kubernetes/HEAD \
   --format='%(objectmode) %(objecttype) %(objectname)%x09%(path)' \
   OWNERS \
   "**/OWNERS"
```

We can generalize to repeat the same for the primary branches of all actively developed remote repositories (i.e. repositories that are not archived in GitHub). This time, we'll only print the git object ID of each `OWNERS` file.

```
f='git ls-files --with-tree=%(refname) --format="%%(objectname)" OWNERS "**/OWNERS"'
git for-each-ref \
   --shell \
   --format="$f" \
   $(gh biome heads) | sh
```

We can output the contents of all the `OWNERS` files as YAML multidoc.

```
f='git ls-files --with-tree=%(refname) --format="%%(objectname)" OWNERS "**/OWNERS"'
git for-each-ref \
   --shell \
   --format="$f" \
   $(gh biome heads) |
sh |
git cat-file --batch=---
```

We can use tools like [`yq`](https://github.com/mikefarah/yq) and [`jq`](https://jqlang.github.io/jq/) to process the `OWNERS` data, ranking users based on how many Kubernetes components they are an approver on.

```
f='git ls-files --with-tree=%(refname) --format="%%(objectname)" OWNERS "**/OWNERS"'
git for-each-ref \
   --shell \
   --format="$f" \
   $(gh biome heads) |
sh |
git cat-file --batch=--- |
yq -o json |
jq -r '.approvers // [] | .[]' |
sort |
uniq -c |
sort -n
```

### Filtering remotes

Often times, there are archived projects in GitHub that we want to exclude from analysis. The biome tracks which remotes are in an active or archived state under git config values. We can list primary branch references for just the active (i.e. non-archived/non-locked) remote GitHub repositories.

```
git for-each-ref $(git config get --all biome.remotes.active | awk '{print "refs/remotes/" $1 "/HEAD"}')
```
or
```
git for-each-ref $(gh biome heads --active)
```
or
```
git for-each-ref $(gh biome heads --active)
```

Or, maybe we care about only the archived projects. We can list primary branch references for archived remote GitHub repositories.

```
git for-each-ref $(git config get --all biome.remotes.archived | awk '{print "refs/remotes/" $1 "/HEAD"}')
```
or
```
git for-each-ref $(gh biome heads --archived)
```

biome tracks the following git config settings for discovered remote repositories:

- `biome.remotes.active` GitHub repositories that are non-archived, non-locked, non-disabled, supported and configured as a git remote on the biome.
- `biome.remotes.archived` GitHub repositories that are archived, no longer actively developed. The repository content is read-only on GitHub. It is configured as a git remote on the biome.
- `biome.remotes.disabled` GitHub repository that has been disabled. Fetches are not supported by GitHub. It is not configured as a git remote.
- `biome.remotes.locked` GitHub repository that has been locked, usually because the repository has been migrated to another GitHub environment, ex. GitHub Enterprise Server to GitHub Enterprise Cloud. Fetches are not supported by GitHub. You should add the repository via its owner in the new GitHub environment instead. It is not configured as a git remote.
- `biome.remotes.unsupported` GitHub repository that is currently unsupported by the biome. In particular, this includes GitHub repositories whose name begins with `.` such as `.github`. It is not configured as a git remote. We'd like to support these in the future.

To list discovered remotes that fall into one or more of these categories, use either `git config get --all biome.remotes.<category>` or `gh biome remotes --<category>`.

```
gh biome remotes --active
gh biome remotes --archived
gh biome remotes --disabled
gh biome remotes --locked
gh biome remotes --unsupported
```

### Updating remotes

To sync our biome with the latest git objects and references from the remotes, including discovery of newly created repositories owned by the GitHub users, we can fetch.

```
gh biome fetch
```

### Have fun

Many more analyses and mutations are possible.

Most underlying git commands support batch-oriented modes where they can operate on many references or objects simultaneously (ex. `git for-each-ref`, `git cat-file --batch{,-check,-command}`, `git rev-list --stdin`, `git update-ref --stdin`, etc). We can use this to our advantage to perform fast bulk operations/queries, amortizing the cost of fetching the git data from the server and avoiding the overhead of opening many individual files on our filesystem. This can often be advantageous over extracting the similar information directly from the GitHub APIs, especially if we're still iterating on the design of our analysis. It also enables analysis even when we lack sufficient disk space to checkout all source files to a local working directory.
