# Harbor Cleaner

Clean images in Harbor by policies.

## Features

- **Delete tags without side effects** As we known when we delete a tag from a repo in docker registry, the underneath manifest is deleted, so are other tags what share the same manifest. In this tool, we protect tags from such situation.
- **Delete by policies** Support delete tags by configurable policies
- **Dry run before actual cleanup** To see what would be cleaned up before performing real cleanup.

## Concepts

| Image | Project | Repo | Tag |
|--|--|--|--|
| library/busybox:1.30.0 | library | busybox | 1.30.0 |  
| release/devops/tools:v1.0 | release | devops/tools | v1.0 |

## Policies

### Number Policy

Number policy will retain latest N tags for each repo and remove other old ones. Latest tags are determined by images' creation time.

```yaml
numberPolicy:
    number: 5
```

This policy takes only one argument, the number of tags to retain.

### Regex Policy

Regex policy removes images that match the given repo and tag regex patterns. A tag will be removed only when following conditions are all satisfied:

- It matches at least one repo pattern
- It matches at least one tag pattern

Regex here are `Golang` supported regex. For example `.*` matches all.

```yaml
regexPolicy:
    repos: [".*"]
    tags: [".*-alpha.*", "dev"]
```

The above policy config will remove tags from all repos that are 'dev' or contain 'alpha'. For example,

- dev
- v1.0.0-alpha
- v1.4.0-alpha.2
- 1.0-alpha.5

## How To Use

### Get Image

```bash
$ make image VERSION=latest
```

You can also pull one from DockerHub.

```bash
$ docker pull k8sdevops/harbor-cleaner:v0.1.1
```

### Configure

```yaml
# Host of the Harbor
host: https://dev.cargo.io
# Version of the Harbor, e.g. 1.7, 1.4.0
version: 1.7
# Admin account
auth:
  user: admin
  password: Pwd123456
# Projects list to clean images for, it you want to clean images for all
# projects, leave it empty.
projects: []
# Policy to clean images
policy:
  # Policy type, e.g. "number", "recentlyNotTouched", "regex"
  type: regex

  # Number policy: to retain the latest N tags for each repo
  #numberPolicy:
  #  number: 5

  # Regex policy: only clean images that match the given repo patterns and tag patterns
  regexPolicy:
    # Regex to match repos, a repo will be regarded as matched when it matches any regex in the list
    repos: [".*"]
    # Regex to match tags, a tag will be regarded as matched when it matches any regex in the list
    tags: [".*-alpha.*", "dev"]

  # Tags that should be retained anyway, '?', '*' supported.
  retainTags: []
```

In the policy part, exact one of `numberPolicy`, `regexPolicy` should be configured according to the policy type. 

### DryRun

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    harbor-cleaner:latest --dryrun=true
```

### Clean

```bash
$ docker run -it --rm \
    -v <your-config-file>:/workspace/config.yaml \
    harbor-cleaner:latest
```

## Supported Version

- 1.4.x
- 1.5.x
- 1.6.x
- 1.7.x
- 1.8.x

It may work for other versions, but this is not tested.